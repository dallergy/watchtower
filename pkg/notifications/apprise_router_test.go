package notifications

import (
	"errors"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Apprise routers", func() {
	AfterEach(func() {
		runCommand = func(name string, args ...string) ([]byte, error) {
			return exec.Command(name, args...).CombinedOutput()
		}
		lookPath = exec.LookPath
	})

	When("non-Gotify notification URLs are configured", func() {
		It("should use the Apprise CLI bundled in this container", func() {
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

			notifier := createNotifier("/etc/apprise.yml", []string{"discord://token/webhook"}, allButTrace, "", true, StaticData{Title: "Watchtower"}, false, time.Duration(0), false)
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
})
