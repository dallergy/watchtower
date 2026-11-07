package auth_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dallergy/watchtower/internal/actions/mocks"
	"github.com/dallergy/watchtower/pkg/registry/auth"

	wtTypes "github.com/dallergy/watchtower/pkg/types"
	ref "github.com/distribution/reference"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registry Auth Suite")
}
func SkipIfCredentialsEmpty(credentials *wtTypes.RegistryCredentials, fn func()) func() {
	if credentials.Username == "" {
		return func() {
			Skip("Username missing. Skipping integration test")
		}
	} else if credentials.Password == "" {
		return func() {
			Skip("Password missing. Skipping integration test")
		}
	} else {
		return fn
	}
}

var GHCRCredentials = &wtTypes.RegistryCredentials{
	Username: os.Getenv("CI_INTEGRATION_TEST_REGISTRY_GH_USERNAME"),
	Password: os.Getenv("CI_INTEGRATION_TEST_REGISTRY_GH_PASSWORD"),
}

var _ = Describe("the auth module", func() {
	mockId := "mock-id"
	mockName := "mock-container"
	mockImage := "ghcr.io/k6io/operator:latest"
	mockCreated := time.Now()
	mockDigest := "ghcr.io/k6io/operator@sha256:d68e1e532088964195ad3a0a71526bc2f11a78de0def85629beb75e2265f0547"

	mockContainer := mocks.CreateMockContainerWithDigest(
		mockId,
		mockName,
		mockImage,
		mockCreated,
		mockDigest)

	Describe("GetToken", func() {
		It("should parse the token from the response",
			SkipIfCredentialsEmpty(GHCRCredentials, func() {
				creds := fmt.Sprintf("%s:%s", GHCRCredentials.Username, GHCRCredentials.Password)
				token, err := auth.GetToken(mockContainer, creds)
				Expect(err).NotTo(HaveOccurred())
				Expect(token).NotTo(Equal(""))
			}),
		)
	})

	Describe("GetAuthURL", func() {
		It("should create a valid auth url object based on the challenge header supplied", func() {
			challenge := `bearer realm="https://ghcr.io/token",service="ghcr.io",scope="repository:user/image:pull"`
			imageRef, err := ref.ParseNormalizedNamed("containrrr/watchtower")
			Expect(err).NotTo(HaveOccurred())
			expected := &url.URL{
				Host:     "ghcr.io",
				Scheme:   "https",
				Path:     "/token",
				RawQuery: "scope=repository%3Acontainrrr%2Fwatchtower%3Apull&service=ghcr.io",
			}

			URL, err := auth.GetAuthURL(challenge, imageRef)
			Expect(err).NotTo(HaveOccurred())
			Expect(URL).To(Equal(expected))
		})

		When("given a challenge header without a realm", func() {
			It("should return an error", func() {
				challenge := `bearer service="ghcr.io"`
				imageRef, err := ref.ParseNormalizedNamed("containrrr/watchtower")
				Expect(err).NotTo(HaveOccurred())
				URL, err := auth.GetAuthURL(challenge, imageRef)
				Expect(err).To(HaveOccurred())
				Expect(URL).To(BeNil())
			})
		})

		When("given a challenge header without a service", func() {
			It("should omit the service from the auth url", func() {
				challenge := `Bearer realm="https://mirror.gcr.io/v2/token"`
				imageRef, err := ref.ParseNormalizedNamed("mirror.gcr.io/library/alpine")
				Expect(err).NotTo(HaveOccurred())
				URL, err := auth.GetAuthURL(challenge, imageRef)
				Expect(err).NotTo(HaveOccurred())
				Expect(URL.String()).To(Equal("https://mirror.gcr.io/v2/token?scope=repository%3Alibrary%2Falpine%3Apull"))
			})
		})

		It("should keep the case of the realm and service", func() {
			challenge := `Bearer realm="https://Registry.example.com/Service/Token?Tenant=ABC",service="Harbor-Registry"`
			imageRef, err := ref.ParseNormalizedNamed("registry.example.com/team/app")
			Expect(err).NotTo(HaveOccurred())
			URL, err := auth.GetAuthURL(challenge, imageRef)
			Expect(err).NotTo(HaveOccurred())
			Expect(URL.Path).To(Equal("/Service/Token"))
			Expect(URL.Query().Get("Tenant")).To(Equal("ABC"))
			Expect(URL.Query().Get("service")).To(Equal("Harbor-Registry"))
		})

		It("should not split quoted values on commas", func() {
			challenge := `Bearer scope="repository:a/b:pull,push",realm="https://auth.example.com/token",service="example"`
			imageRef, err := ref.ParseNormalizedNamed("example.com/a/b")
			Expect(err).NotTo(HaveOccurred())
			URL, err := auth.GetAuthURL(challenge, imageRef)
			Expect(err).NotTo(HaveOccurred())
			Expect(URL.Host).To(Equal("auth.example.com"))
			Expect(URL.Query().Get("service")).To(Equal("example"))
		})

		When("deriving the auth scope from an image name", func() {
			It("should prepend official dockerhub images with \"library/\"", func() {
				Expect(getScopeFromImageAuthURL("registry")).To(Equal("library/registry"))
				Expect(getScopeFromImageAuthURL("docker.io/registry")).To(Equal("library/registry"))
				Expect(getScopeFromImageAuthURL("index.docker.io/registry")).To(Equal("library/registry"))
			})
			It("should not include vanity hosts\"", func() {
				Expect(getScopeFromImageAuthURL("docker.io/containrrr/watchtower")).To(Equal("containrrr/watchtower"))
				Expect(getScopeFromImageAuthURL("index.docker.io/containrrr/watchtower")).To(Equal("containrrr/watchtower"))
			})
			It("should not destroy three segment image names\"", func() {
				Expect(getScopeFromImageAuthURL("piksel/containrrr/watchtower")).To(Equal("piksel/containrrr/watchtower"))
				Expect(getScopeFromImageAuthURL("ghcr.io/piksel/containrrr/watchtower")).To(Equal("piksel/containrrr/watchtower"))
			})
			It("should not prepend library/ to image names if they're not on dockerhub", func() {
				Expect(getScopeFromImageAuthURL("ghcr.io/watchtower")).To(Equal("watchtower"))
				Expect(getScopeFromImageAuthURL("ghcr.io/containrrr/watchtower")).To(Equal("containrrr/watchtower"))
			})
		})
		It("should not crash when an empty field is received", func() {
			input := `bearer realm="https://ghcr.io/token",service="ghcr.io",scope="repository:user/image:pull",`
			imageRef, err := ref.ParseNormalizedNamed("containrrr/watchtower")
			Expect(err).NotTo(HaveOccurred())
			res, err := auth.GetAuthURL(input, imageRef)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
		})
		It("should not crash when a field without a value is received", func() {
			input := `bearer realm="https://ghcr.io/token",service="ghcr.io",scope="repository:user/image:pull",valuelesskey`
			imageRef, err := ref.ParseNormalizedNamed("containrrr/watchtower")
			Expect(err).NotTo(HaveOccurred())
			res, err := auth.GetAuthURL(input, imageRef)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
		})
	})

	Describe("GetBearerHeader", func() {
		var server *httptest.Server
		var requests int
		var response string
		var status int
		imageRef, _ := ref.ParseNormalizedNamed("registry.example.com/team/app")

		BeforeEach(func() {
			requests = 0
			status = http.StatusOK
			response = `{"token":"abc"}`
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.WriteHeader(status)
				_, _ = w.Write([]byte(response))
			}))
		})
		AfterEach(func() {
			server.Close()
		})

		challenge := func() string {
			return fmt.Sprintf(`Bearer realm="%s/token",service="registry"`, server.URL)
		}

		It("should return the token", func() {
			header, err := auth.GetBearerHeader(challenge(), imageRef, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(header).To(Equal("Bearer abc"))
		})
		It("should accept an access_token instead of a token", func() {
			response = `{"access_token":"def"}`
			header, err := auth.GetBearerHeader(challenge(), imageRef, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(header).To(Equal("Bearer def"))
		})
		It("should fail when the token service rejects the request", func() {
			status = http.StatusUnauthorized
			response = `{"errors":[]}`
			_, err := auth.GetBearerHeader(challenge(), imageRef, "")
			Expect(err).To(MatchError(ContainSubstring("401 Unauthorized")))
		})
		It("should fail when no token is returned", func() {
			response = `{}`
			_, err := auth.GetBearerHeader(challenge(), imageRef, "")
			Expect(err).To(MatchError(ContainSubstring("did not include a token")))
		})
		It("should not send credentials to a token service without HTTPS", func() {
			_, err := auth.GetBearerHeader(challenge(), imageRef, "dXNlcjpwYXNz")
			Expect(err).To(MatchError(ContainSubstring("does not use HTTPS")))
			Expect(requests).To(Equal(0))
		})
	})

	Describe("GetChallengeURL", func() {
		It("should create a valid challenge url object based on the image ref supplied", func() {
			expected := url.URL{Host: "ghcr.io", Scheme: "https", Path: "/v2/"}
			imageRef, _ := ref.ParseNormalizedNamed("ghcr.io/containrrr/watchtower:latest")
			Expect(auth.GetChallengeURL(imageRef)).To(Equal(expected))
		})
		It("should assume Docker Hub for image refs with no explicit registry", func() {
			expected := url.URL{Host: "index.docker.io", Scheme: "https", Path: "/v2/"}
			imageRef, _ := ref.ParseNormalizedNamed("containrrr/watchtower:latest")
			Expect(auth.GetChallengeURL(imageRef)).To(Equal(expected))
		})
		It("should use index.docker.io if the image ref specifies docker.io", func() {
			expected := url.URL{Host: "index.docker.io", Scheme: "https", Path: "/v2/"}
			imageRef, _ := ref.ParseNormalizedNamed("docker.io/containrrr/watchtower:latest")
			Expect(auth.GetChallengeURL(imageRef)).To(Equal(expected))
		})
	})
})

var scopeImageRegexp = MatchRegexp("^repository:[a-z0-9]+(/[a-z0-9]+)*:pull$")

func getScopeFromImageAuthURL(imageName string) string {
	normalizedRef, _ := ref.ParseNormalizedNamed(imageName)
	challenge := `bearer realm="https://dummy.host/token",service="dummy.host",scope="repository:user/image:pull"`
	URL, _ := auth.GetAuthURL(challenge, normalizedRef)

	scope := URL.Query().Get("scope")
	Expect(scopeImageRegexp.Match(scope)).To(BeTrue())
	return strings.Replace(scope[11:], ":pull", "", 1)
}
