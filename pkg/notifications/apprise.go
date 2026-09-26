package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/dallergy/watchtower/pkg/notifications/apprise"
	"github.com/dallergy/watchtower/pkg/notifications/templates"
	t "github.com/dallergy/watchtower/pkg/types"
	log "github.com/sirupsen/logrus"
)

// LocalLog is a logrus logger that does not send entries as notifications
var LocalLog = log.WithField("notify", "no")

const (
	appriseType     = "apprise"
	stdoutScheme    = "stdout"
	legacyStdoutURL = "logger://"
	// sendTimeout limits how long delivering a single notification to all services may take
	sendTimeout = 2 * time.Minute
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

type appriseNotificationRequest struct {
	URLs   []string `json:"urls,omitempty"`
	Body   string   `json:"body"`
	Title  string   `json:"title,omitempty"`
	Format string   `json:"format"`
	Type   string   `json:"type"`
}

type httpAppriseRouter struct {
	client     *http.Client
	notifyURL  string
	urls       []string
	useKeyMode bool
}

func newHTTPAppriseRouter(appriseURL, appriseKey string, urls []string) *httpAppriseRouter {
	baseURL := strings.TrimRight(appriseURL, "/")
	notifyURL := baseURL + "/notify/"
	if appriseKey != "" {
		notifyURL = baseURL + "/notify/" + appriseKey
	}

	return &httpAppriseRouter{
		client:     &http.Client{Timeout: 30 * time.Second},
		notifyURL:  notifyURL,
		urls:       urls,
		useKeyMode: appriseKey != "",
	}
}

// ServiceError is a failure to deliver a notification to one of the configured services
type ServiceError struct {
	Service string
	Err     error
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %v", e.Service, e.Err)
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}

func (r *httpAppriseRouter) Send(message string, params *notificationParams) []error {
	reqBody := appriseNotificationRequest{
		Body:   message,
		Format: "text",
		Type:   "info",
	}

	if !r.useKeyMode {
		reqBody.URLs = r.urls
	}

	if params != nil {
		if title, ok := params.Title(); ok {
			reqBody.Title = title
		}
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return []error{fmt.Errorf("failed to marshal notification request: %w", err)}
	}

	req, err := http.NewRequest(http.MethodPost, r.notifyURL, bytes.NewReader(payload))
	if err != nil {
		return []error{fmt.Errorf("failed to create notification request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return []error{&ServiceError{Service: appriseType, Err: fmt.Errorf("failed to reach the Apprise API: %w", err)}}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return []error{&ServiceError{Service: appriseType, Err: fmt.Errorf("the Apprise API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))}}
	}

	return nil
}

// serviceRouter delivers notifications to the natively supported services
type serviceRouter struct {
	services []apprise.Service
}

func newServiceRouter(urls []string, opts apprise.Options) (*serviceRouter, error) {
	services := make([]apprise.Service, 0, len(urls))
	for _, u := range urls {
		service, err := apprise.Parse(u, opts)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	return &serviceRouter{services: services}, nil
}

func (r *serviceRouter) Send(message string, params *notificationParams) []error {
	msg := apprise.Message{Body: message}
	if title, ok := params.Title(); ok {
		msg.Title = title
	}

	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()

	var errs []error
	for _, service := range r.services {
		if err := service.Send(ctx, msg); err != nil {
			errs = append(errs, &ServiceError{Service: service.Scheme(), Err: err})
		}
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

func createNotifier(appriseURL, appriseKey, appriseConfig string, urls []string, level log.Level, tplString string, legacy bool, data StaticData, stdout bool, delay time.Duration, gotifySkipVerify bool) *appriseTypeNotifier {
	tpl, err := getNotificationTemplate(tplString, legacy)
	if err != nil {
		log.Errorf("Could not use configured notification template: %s. Using default template", err)
	}

	params := &notificationParams{}
	if data.Title != "" {
		params.title = data.Title
	}

	if appriseConfig != "" {
		configURLs, err := apprise.LoadConfig(appriseConfig)
		if err != nil {
			log.Fatal("Failed to load the Apprise config: ", err)
		}
		urls = append(urls, configURLs...)
	}

	r, err := createRouter(appriseURL, appriseKey, urls, stdout, gotifySkipVerify)
	if err != nil {
		log.Fatal("Failed to initialize notifications: ", err)
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

// createRouter sends notifications for built in services natively, while other services are
// forwarded to an external Apprise API (when configured)
func createRouter(appriseURL, appriseKey string, urls []string, stdout bool, gotifySkipVerify bool) (router, error) {
	if usesStdoutOnly(urls, stdout) {
		return newStdoutRouter(stdout), nil
	}

	var nativeURLs, forwardedURLs []string
	for _, u := range filterNotificationURLs(urls) {
		switch {
		case apprise.IsSupported(u):
			if gotifySkipVerify {
				u = withTLSVerifyDisabled(u)
			}
			if apprise.Scheme(u) == "gotify" {
				// earlier releases of this fork used HTTPS for gotify:// URLs with a public host name
				LocalLog.Warn("gotify:// notifications are sent over plain HTTP, use gotifys:// for servers that use HTTPS")
			}
			nativeURLs = append(nativeURLs, u)
		case appriseURL != "":
			forwardedURLs = append(forwardedURLs, u)
		default:
			scheme := apprise.Scheme(u)
			return nil, fmt.Errorf("%w; set --notification-apprise-url to forward it to an Apprise API server",
				apprise.UnsupportedError{Scheme: scheme})
		}
	}

	var routers []router
	if hasLegacyStdoutURL(urls) {
		routers = append(routers, newStdoutRouter(false))
	}
	if len(nativeURLs) > 0 {
		native, err := newServiceRouter(nativeURLs, apprise.Options{})
		if err != nil {
			return nil, err
		}
		routers = append(routers, native)
	}
	if appriseURL != "" && (len(forwardedURLs) > 0 || appriseKey != "") {
		routers = append(routers, newHTTPAppriseRouter(appriseURL, appriseKey, forwardedURLs))
	}

	return combineRouters(routers...), nil
}

// withTLSVerifyDisabled applies the legacy Gotify TLS flag to Gotify URLs
func withTLSVerifyDisabled(u string) string {
	scheme := apprise.Scheme(u)
	if (scheme != "gotify" && scheme != "gotifys") || strings.Contains(u, "verify=") {
		return u
	}
	separator := "?"
	if strings.Contains(u, "?") {
		separator = "&"
	}
	return u + separator + "verify=no"
}

func hasLegacyStdoutURL(urls []string) bool {
	for _, u := range urls {
		if strings.TrimSpace(u) == legacyStdoutURL || apprise.Scheme(u) == stdoutScheme {
			return true
		}
	}
	return false
}

type fanoutRouter struct {
	children []router
}

func (r *fanoutRouter) Send(message string, params *notificationParams) []error {
	var errs []error
	for _, child := range r.children {
		errs = append(errs, child.Send(message, params)...)
	}
	return errs
}

func combineRouters(routers ...router) router {
	switch len(routers) {
	case 0:
		return &noopRouter{}
	case 1:
		return routers[0]
	default:
		return &fanoutRouter{children: routers}
	}
}

func sendNotifications(n *appriseTypeNotifier) {
	for msg := range n.messages {
		time.Sleep(n.delay)
		errs := n.Router.Send(msg, n.params)

		for _, err := range errs {
			if err == nil {
				continue
			}
			fields := log.Fields{}
			var serviceErr *ServiceError
			if errors.As(err, &serviceErr) {
				fields["service"] = serviceErr.Service
				err = serviceErr.Err
			}
			LocalLog.WithFields(fields).WithError(err).Error("Failed to send notification")
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
