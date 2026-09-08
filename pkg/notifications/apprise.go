package notifications

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/containrrr/watchtower/pkg/notifications/templates"
	t "github.com/containrrr/watchtower/pkg/types"
	log "github.com/sirupsen/logrus"
)

// LocalLog is a logrus logger that does not send entries as notifications
var LocalLog = log.WithField("notify", "no")

const (
	appriseType     = "apprise"
	stdoutScheme    = "stdout"
	legacyStdoutURL = "logger://"
	appriseBin      = "apprise"
)

type notificationParams struct {
	title string
}

func (p *notificationParams) Title() (string, bool) {
	if p == nil || p.title == "" {
		return "", false
	}
	return p.title, true
}

type router interface {
	Send(message string, params *notificationParams) []error
}

type execRunner func(name string, args ...string) ([]byte, error)
type lookPather func(file string) (string, error)

var runCommand execRunner = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

var lookPath lookPather = exec.LookPath

// Implements Notifier, logrus.Hook
type appriseTypeNotifier struct {
	Urls           []string
	Router         router
	entries        []*log.Entry
	logLevel       log.Level
	template       *template.Template
	messages       chan string
	done           chan bool
	legacyTemplate bool
	params         *notificationParams
	data           StaticData
	receiving      bool
	delay          time.Duration
}

// GetScheme returns the scheme part of a notification URL
func GetScheme(url string) string {
	schemeEnd := strings.Index(url, ":")
	if schemeEnd <= 0 {
		return "invalid"
	}
	return url[:schemeEnd]
}

// GetNames returns a list of notification services that has been added
func (n *appriseTypeNotifier) GetNames() []string {
	names := make([]string, len(n.Urls))
	for i, u := range n.Urls {
		names[i] = GetScheme(u)
	}
	return names
}

// GetURLs returns a list of URLs for notification services that has been added
func (n *appriseTypeNotifier) GetURLs() []string {
	return n.Urls
}

// AddLogHook adds the notifier as a receiver of log messages and starts a go func for processing them
func (n *appriseTypeNotifier) AddLogHook() {
	if n.receiving {
		return
	}
	n.receiving = true
	log.AddHook(n)

	go sendNotifications(n)
}

func perURLErrors(urls []string, err error) []error {
	if len(urls) == 0 {
		return []error{err}
	}
	errs := make([]error, len(urls))
	for i := range urls {
		errs[i] = err
	}
	return errs
}

type noopRouter struct{}

func (r *noopRouter) Send(_ string, _ *notificationParams) []error {
	return nil
}

type stdoutRouter struct {
	writer io.Writer
}

func newStdoutRouter(stdout bool) *stdoutRouter {
	if stdout {
		return &stdoutRouter{writer: os.Stdout}
	}
	return &stdoutRouter{writer: log.StandardLogger().WriterLevel(log.TraceLevel)}
}

func (r *stdoutRouter) Send(message string, params *notificationParams) []error {
	if params != nil {
		if title, ok := params.Title(); ok {
			_, err := fmt.Fprintf(r.writer, "%s\n%s\n", title, message)
			if err != nil {
				return []error{err}
			}
			return nil
		}
	}
	_, err := fmt.Fprintln(r.writer, message)
	if err != nil {
		return []error{err}
	}
	return nil
}

type cliAppriseRouter struct {
	urls   []string
	config string
	bin    string
	run    execRunner
}

func newCLIAppriseRouter(urls []string, config string) *cliAppriseRouter {
	return &cliAppriseRouter{
		urls:   urls,
		config: config,
		bin:    appriseBin,
		run:    runCommand,
	}
}

func (r *cliAppriseRouter) Send(message string, params *notificationParams) []error {
	args := []string{"--input-format", "text", "--notification-type", "info", "--body", message}
	if params != nil {
		if title, ok := params.Title(); ok {
			args = append(args, "--title", title)
		}
	}
	if r.config != "" {
		args = append(args, "--config", r.config)
	}
	args = append(args, r.urls...)

	out, err := r.run(r.bin, args...)
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail == "" {
			detail = err.Error()
		} else {
			detail = fmt.Sprintf("%s: %s", err.Error(), detail)
		}
		return perURLErrors(r.urls, fmt.Errorf("failed to send apprise notification: %s", detail))
	}

	return make([]error, len(r.urls))
}

func filterNotificationURLs(urls []string) []string {
	filtered := make([]string, 0, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		scheme := GetScheme(u)
		if scheme == stdoutScheme || u == legacyStdoutURL {
			continue
		}
		filtered = append(filtered, u)
	}
	return filtered
}

func usesStdoutOnly(urls []string, stdout bool) bool {
	if stdout {
		return true
	}
	if len(urls) == 0 {
		return false
	}
	for _, u := range urls {
		scheme := GetScheme(u)
		if scheme != stdoutScheme && u != legacyStdoutURL {
			return false
		}
	}
	return true
}

func createNotifier(appriseConfig string, urls []string, level log.Level, tplString string, legacy bool, data StaticData, stdout bool, delay time.Duration, gotifySkipVerify bool) *appriseTypeNotifier {
	tpl, err := getNotificationTemplate(tplString, legacy)
	if err != nil {
		log.Errorf("Could not use configured notification template: %s. Using default template", err)
	}

	params := &notificationParams{}
	if data.Title != "" {
		params.title = data.Title
	}

	var r router
	if usesStdoutOnly(urls, stdout) {
		r = newStdoutRouter(stdout)
	} else {
		serviceURLs := filterNotificationURLs(urls)
		gotifyURLs, otherURLs := splitGotifyURLs(serviceURLs)

		var gotifyRouter router
		if len(gotifyURLs) > 0 {
			gotifyRouter, err = newGotifyHTTPRouter(gotifyURLs, gotifySkipVerify)
			if err != nil {
				log.Fatal("Failed to initialize Gotify notifications: ", err)
			}
		}

		var appriseRouter router
		hasOther := len(otherURLs) > 0 || appriseConfig != ""
		if hasOther {
			if _, err := lookPath(appriseBin); err != nil {
				log.Fatal("Failed to initialize extra notification URLs: install/use the official Watchtower image, which bundles Apprise in the same container. Gotify does not need Apprise.")
			}
			appriseRouter = newCLIAppriseRouter(otherURLs, appriseConfig)
		}

		r = combineRouters(gotifyRouter, appriseRouter)
	}

	return &appriseTypeNotifier{
		Urls:           urls,
		Router:         r,
		messages:       make(chan string, 1),
		done:           make(chan bool),
		logLevel:       level,
		template:       tpl,
		legacyTemplate: legacy,
		data:           data,
		params:         params,
		delay:          delay,
	}
}

func sendNotifications(n *appriseTypeNotifier) {
	for msg := range n.messages {
		time.Sleep(n.delay)
		errs := n.Router.Send(msg, n.params)

		for i, err := range errs {
			if err != nil {
				scheme := "apprise"
				if i < len(n.Urls) {
					scheme = GetScheme(n.Urls[i])
				}
				LocalLog.WithFields(log.Fields{
					"service": scheme,
					"index":   i,
				}).WithError(err).Error("Failed to send apprise notification")
			}
		}
	}

	n.done <- true
}

func (n *appriseTypeNotifier) buildMessage(data Data) (string, error) {
	var body bytes.Buffer
	var templateData interface{} = data
	if n.legacyTemplate {
		templateData = data.Entries
	}
	if err := n.template.Execute(&body, templateData); err != nil {
		return "", err
	}

	return body.String(), nil
}

func (n *appriseTypeNotifier) sendEntries(entries []*log.Entry, report t.Report) {
	msg, err := n.buildMessage(Data{n.data, entries, report})

	if msg == "" {
		go func() {
			if err != nil {
				LocalLog.WithError(err).Fatal("Notification template error")
			} else if len(n.Urls) > 1 {
				LocalLog.Info("Skipping notification due to empty message")
			}
		}()
		return
	}
	n.messages <- msg
}

// StartNotification begins queueing up messages to send them as a batch
func (n *appriseTypeNotifier) StartNotification() {
	if n.entries == nil {
		n.entries = make([]*log.Entry, 0, 10)
	}
}

// SendNotification sends the queued up messages as a batch
func (n *appriseTypeNotifier) SendNotification(report t.Report) {
	n.sendEntries(n.entries, report)
	n.entries = nil
}

// Close prevents further messages from being queued and waits until all the currently queued up messages have been sent
func (n *appriseTypeNotifier) Close() {
	close(n.messages)

	LocalLog.Info("Waiting for the notification goroutine to finish")

	<-n.done
}

// Levels return what log levels trigger notifications
func (n *appriseTypeNotifier) Levels() []log.Level {
	return log.AllLevels[:n.logLevel+1]
}

// Fire is the hook that logrus calls on a new log message
func (n *appriseTypeNotifier) Fire(entry *log.Entry) error {
	if entry.Data["notify"] == "no" {
		return nil
	}
	if n.entries != nil {
		n.entries = append(n.entries, entry)
	} else {
		n.sendEntries([]*log.Entry{entry}, nil)
	}
	return nil
}

func getNotificationTemplate(tplString string, legacy bool) (tpl *template.Template, err error) {
	tplBase := template.New("").Funcs(templates.Funcs)

	if builtin, found := commonTemplates[tplString]; found {
		log.WithField(`template`, tplString).Debug(`Using common template`)
		tplString = builtin
	}

	if tplString != "" {
		tpl, err = tplBase.Parse(tplString)
	}

	if err != nil || tplString == "" {
		defaultKey := `default`
		if legacy {
			defaultKey = `default-legacy`
		}

		tpl = template.Must(tplBase.Parse(commonTemplates[defaultKey]))
	}

	return
}
