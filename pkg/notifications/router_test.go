package notifications

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

// requestLog records the requests received by a test server
type requestLog struct {
	mu       sync.Mutex
	paths    []string
	bodies   []string
	gotified []string
}

func (l *requestLog) handler(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		l.mu.Lock()
		l.paths = append(l.paths, r.URL.Path)
		l.bodies = append(l.bodies, string(body))
		l.gotified = append(l.gotified, r.Header.Get("X-Gotify-Key"))
		l.mu.Unlock()
		w.WriteHeader(status)
	}
}

var _ = Describe("notification routers", func() {
	var requests *requestLog

	BeforeEach(func() {
		requests = &requestLog{}
	})

	When("only built in services are configured", func() {
		It("should deliver notifications natively", func() {
			server := httptest.NewServer(requests.handler(http.StatusOK))
			defer server.Close()
			gotifyURL := fmt.Sprintf("gotify://%s/app-token", strings.TrimPrefix(server.URL, "http://"))

			notifier := createNotifier("", "", "", []string{gotifyURL}, logrus.InfoLevel, "", true, StaticData{Title: "Watchtower"}, false, time.Duration(0), false)
			Expect(notifier.Router).To(BeAssignableToTypeOf(&serviceRouter{}))

			Expect(notifier.Router.Send("updated", notifier.params)).To(BeEmpty())
			Expect(requests.paths).To(Equal([]string{"/message"}))
			Expect(requests.gotified).To(Equal([]string{"app-token"}))
			Expect(requests.bodies[0]).To(ContainSubstring(`"title":"Watchtower"`))
			Expect(requests.bodies[0]).To(ContainSubstring(`"message":"updated"`))
		})

		It("should report failures per service", func() {
			server := httptest.NewServer(requests.handler(http.StatusUnauthorized))
			defer server.Close()
			gotifyURL := fmt.Sprintf("gotify://%s/app-token", strings.TrimPrefix(server.URL, "http://"))

			notifier := createNotifier("", "", "", []string{gotifyURL}, logrus.InfoLevel, "", true, StaticData{}, false, time.Duration(0), false)
			errs := notifier.Router.Send("updated", notifier.params)
			Expect(errs).To(HaveLen(1))
			var serviceErr *ServiceError
			Expect(errs[0]).To(BeAssignableToTypeOf(serviceErr))
			Expect(errs[0].(*ServiceError).Service).To(Equal("gotify"))
			Expect(errs[0].Error()).To(ContainSubstring("401 Unauthorized"))
		})
	})

	When("the legacy gotify TLS flag is set", func() {
		It("should disable certificate verification for gotify URLs only", func() {
			Expect(withTLSVerifyDisabled("gotifys://host/token")).To(Equal("gotifys://host/token?verify=no"))
			Expect(withTLSVerifyDisabled("gotifys://host/token?priority=high")).To(Equal("gotifys://host/token?priority=high&verify=no"))
			Expect(withTLSVerifyDisabled("gotifys://host/token?verify=yes")).To(Equal("gotifys://host/token?verify=yes"))
			Expect(withTLSVerifyDisabled("ntfys://host/topic")).To(Equal("ntfys://host/topic"))
		})
	})

	When("a service is not built in", func() {
		It("should fail without an Apprise API", func() {
			_, err := createRouter("", "", []string{"matrix://user:pass@example.com/#room"}, false, false)
			Expect(err).To(MatchError(ContainSubstring(`"matrix" notification service is not built in`)))
			Expect(err).To(MatchError(ContainSubstring("--notification-apprise-url")))
		})

		It("should only forward the unsupported services to the Apprise API", func() {
			var got appriseNotificationRequest
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer GinkgoRecover()
				Expect(r.Method).To(Equal(http.MethodPost))
				Expect(r.URL.Path).To(Equal("/notify/"))
				body, err := io.ReadAll(r.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(body, &got)).To(Succeed())
				w.WriteHeader(http.StatusOK)
			}))
			defer api.Close()
			gotify := httptest.NewServer(requests.handler(http.StatusOK))
			defer gotify.Close()
			gotifyURL := fmt.Sprintf("gotify://%s/app-token", strings.TrimPrefix(gotify.URL, "http://"))

			urls := []string{"matrix://user:pass@example.com/#room", gotifyURL}
			notifier := createNotifier(api.URL, "", "", urls, logrus.InfoLevel, "", true, StaticData{Title: "Watchtower"}, false, time.Duration(0), false)
			Expect(notifier.Router).To(BeAssignableToTypeOf(&fanoutRouter{}))

			Expect(notifier.Router.Send("updated", notifier.params)).To(BeEmpty())
			Expect(got.Body).To(Equal("updated"))
			Expect(got.Title).To(Equal("Watchtower"))
			Expect(got.URLs).To(Equal([]string{"matrix://user:pass@example.com/#room"}))
			Expect(requests.paths).To(Equal([]string{"/message"}))
		})
	})

	When("an Apprise config file is given", func() {
		It("should add the URLs from the file", func() {
			server := httptest.NewServer(requests.handler(http.StatusOK))
			defer server.Close()
			host := strings.TrimPrefix(server.URL, "http://")

			configPath := filepath.Join(GinkgoT().TempDir(), "apprise.yml")
			config := fmt.Sprintf("urls:\n  - gotify://%s/one\n  - json://%s/two\n", host, host)
			Expect(os.WriteFile(configPath, []byte(config), 0o600)).To(Succeed())

			notifier := createNotifier("", "", configPath, nil, logrus.InfoLevel, "", true, StaticData{}, false, time.Duration(0), false)
			Expect(notifier.GetNames()).To(Equal([]string{"gotify", "json"}))
			Expect(notifier.Router.Send("updated", notifier.params)).To(BeEmpty())
			Expect(requests.paths).To(Equal([]string{"/message", "/two"}))
		})
	})

	When("logger:// is combined with other services", func() {
		It("should keep writing notifications to the log", func() {
			router, err := createRouter("", "", []string{"logger://", "gotify://gotify.local/token"}, false, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(router).To(BeAssignableToTypeOf(&fanoutRouter{}))
			Expect(router.(*fanoutRouter).children[0]).To(BeAssignableToTypeOf(&stdoutRouter{}))
		})
	})
})
