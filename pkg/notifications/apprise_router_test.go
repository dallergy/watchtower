package notifications

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Apprise routers", func() {
	AfterEach(func() {
		runCommand = func(name string, args ...string) ([]byte, error) {
			return exec.Command(name, args...).CombinedOutput()
		}
		lookPath = exec.LookPath
	})

	When("notification URLs are configured without an external API", func() {
		It("should use the bundled Apprise CLI", func() {
			lookPath = func(file string) (string, error) {
				Expect(file).To(Equal("apprise"))
				return "/usr/bin/apprise", nil
			}

			var capturedName string
			var capturedArgs []string
			runCommand = func(name string, args ...string) ([]byte, error) {
				capturedName = name
				capturedArgs = append([]string{}, args...)
				return []byte("ok"), nil
			}

			notifier := createNotifier("", "", "/etc/apprise.yml", []string{"discord://token/webhook"}, allButTrace, "", true, StaticData{Title: "Watchtower"}, false, time.Duration(0), false)
			Expect(notifier.Router).To(BeAssignableToTypeOf(&cliAppriseRouter{}))

			errs := notifier.Router.Send("hello world", notifier.params)
			Expect(errs).To(HaveLen(1))
			Expect(errs[0]).To(BeNil())
			Expect(capturedName).To(Equal("apprise"))
			Expect(capturedArgs).To(ContainElement("--body"))
			Expect(capturedArgs).To(ContainElement("hello world"))
			Expect(capturedArgs).To(ContainElement("--title"))
			Expect(capturedArgs).To(ContainElement("Watchtower"))
			Expect(capturedArgs).To(ContainElement("--config"))
			Expect(capturedArgs).To(ContainElement("/etc/apprise.yml"))
			Expect(capturedArgs).To(ContainElement("discord://token/webhook"))
		})

		It("should surface CLI failures", func() {
			lookPath = func(string) (string, error) {
				return "/usr/bin/apprise", nil
			}
			runCommand = func(string, ...string) ([]byte, error) {
				return []byte("connection refused"), errors.New("exit status 1")
			}

			router := newCLIAppriseRouter([]string{"mailto://user:pass@localhost"}, "")
			errs := router.Send("body", nil)
			Expect(errs).To(HaveLen(1))
			Expect(errs[0]).To(HaveOccurred())
			Expect(errs[0].Error()).To(ContainSubstring("connection refused"))
		})
	})

	When("an external Apprise API URL is configured", func() {
		It("should POST to the bundled-compatible /notify/ endpoint", func() {
			var got appriseNotificationRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal(http.MethodPost))
				Expect(r.URL.Path).To(Equal("/notify/"))
				body, err := io.ReadAll(r.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(body, &got)).To(Succeed())
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			lookPath = func(string) (string, error) {
				Fail("CLI should not be used when an API URL is set")
				return "", errors.New("not used")
			}

			notifier := createNotifier(server.URL, "", "", []string{"slack://a/b/c"}, logrus.InfoLevel, "", true, StaticData{Title: "Watchtower"}, false, time.Duration(0), false)
			Expect(notifier.Router).To(BeAssignableToTypeOf(&httpAppriseRouter{}))

			errs := notifier.Router.Send("updated", notifier.params)
			Expect(errs).To(HaveLen(1))
			Expect(errs[0]).To(BeNil())
			Expect(got.Body).To(Equal("updated"))
			Expect(got.Title).To(Equal("Watchtower"))
			Expect(got.URLs).To(Equal([]string{"slack://a/b/c"}))
		})
	})
})
