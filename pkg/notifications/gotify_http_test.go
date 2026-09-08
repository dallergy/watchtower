package notifications

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Gotify HTTP router", func() {
	AfterEach(func() {
		lookPath = exec.LookPath
	})

	It("should POST JSON to /message without using Apprise", func() {
		var gotPath, gotToken, gotContentType string
		var payload gotifyPayload
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotToken = r.URL.Query().Get("token")
			gotContentType = r.Header.Get("Content-Type")
			body, err := io.ReadAll(r.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(body, &payload)).To(Succeed())
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":1}`))
		}))
		defer server.Close()

		lookPath = func(string) (string, error) {
			Fail("Gotify must not require the Apprise CLI")
			return "", nil
		}

		host := server.Listener.Addr().String()
		notifier := createNotifier("", []string{"gotify://" + host + "/gtfy.token"}, allButTrace, "", true, StaticData{Title: "Watchtower updates"}, false, time.Duration(0), false)
		Expect(notifier.Router).To(BeAssignableToTypeOf(&gotifyHTTPRouter{}))

		errs := notifier.Router.Send("container nginx updated", notifier.params)
		Expect(errs).To(HaveLen(1))
		Expect(errs[0]).NotTo(HaveOccurred())
		Expect(gotPath).To(Equal("/message"))
		Expect(gotToken).To(Equal("gtfy.token"))
		Expect(gotContentType).To(Equal("application/json"))
		Expect(payload.Title).To(Equal("Watchtower updates"))
		Expect(payload.Message).To(Equal("container nginx updated"))
		Expect(payload.Priority).To(Equal(5))
	})

	It("should use HTTPS for gotify:// on a public hostname", func() {
		endpoint, err := parseGotifyAppriseURL("gotify://notify.example.com/app-token")
		Expect(err).NotTo(HaveOccurred())
		Expect(endpoint.messageURL).To(Equal("https://notify.example.com/message?token=app-token"))
	})

	It("should preserve a Gotify subpath", func() {
		endpoint, err := parseGotifyAppriseURL("gotifys://notify.example.com/gotify/app-token")
		Expect(err).NotTo(HaveOccurred())
		Expect(endpoint.host).To(Equal("notify.example.com"))
		Expect(endpoint.messageURL).To(Equal("https://notify.example.com/gotify/message?token=app-token"))
	})

	It("should convert https Gotify servers to gotifys:// then back to https /message", func() {
		n := &gotifyTypeNotifier{
			gotifyURL:      "https://notify.example.com/",
			gotifyAppToken: "app-token",
		}
		raw, err := n.GetURL(nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(raw).To(Equal("gotifys://notify.example.com/app-token"))

		endpoint, err := parseGotifyAppriseURL(raw)
		Expect(err).NotTo(HaveOccurred())
		Expect(endpoint.messageURL).To(Equal("https://notify.example.com/message?token=app-token"))
	})
})
