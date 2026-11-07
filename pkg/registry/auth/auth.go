package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/dallergy/watchtower/pkg/registry/helpers"
	"github.com/dallergy/watchtower/pkg/types"
	ref "github.com/distribution/reference"
	"github.com/sirupsen/logrus"
)

// ChallengeHeader is the HTTP Header containing challenge instructions
const ChallengeHeader = "WWW-Authenticate"

// maxTokenResponseSize limits how much of a token response is read
const maxTokenResponseSize = 1 << 20

// GetToken fetches a token for the registry hosting the provided image
func GetToken(container types.Container, registryAuth string) (string, error) {
	normalizedRef, err := ref.ParseNormalizedNamed(container.ImageName())
	if err != nil {
		return "", err
	}

	URL := GetChallengeURL(normalizedRef)
	logrus.WithField("URL", URL.String()).Debug("Built challenge URL")

	var req *http.Request
	if req, err = GetChallengeRequest(URL); err != nil {
		return "", err
	}

	var res *http.Response
	if res, err = helpers.HTTPClient().Do(req); err != nil {
		return "", err
	}
	_ = res.Body.Close()
	v := res.Header.Get(ChallengeHeader)

	logrus.WithFields(logrus.Fields{
		"status": res.Status,
		"header": v,
	}).Debug("Got response to challenge request")

	scheme, _ := parseChallenge(v)
	switch scheme {
	case "basic":
		if registryAuth == "" {
			return "", fmt.Errorf("no credentials available")
		}
		return fmt.Sprintf("Basic %s", registryAuth), nil
	case "bearer":
		return GetBearerHeader(v, normalizedRef, registryAuth)
	}

	return "", errors.New("unsupported challenge type from registry")
}

// GetChallengeRequest creates a request for getting challenge instructions
func GetChallengeRequest(URL url.URL) (*http.Request, error) {
	req, err := http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "Watchtower (Docker)")
	return req, nil
}

// GetBearerHeader tries to fetch a bearer token from the registry based on the challenge instructions
func GetBearerHeader(challenge string, imageRef ref.Named, registryAuth string) (string, error) {
	authURL, err := GetAuthURL(challenge, imageRef)

	if err != nil {
		return "", err
	}

	var r *http.Request
	if r, err = http.NewRequest("GET", authURL.String(), nil); err != nil {
		return "", err
	}

	if registryAuth != "" {
		if authURL.Scheme != "https" {
			return "", fmt.Errorf("refusing to send registry credentials to the token service at %s, as it does not use HTTPS", authURL.Host)
		}
		logrus.Debug("Credentials found.")
		// CREDENTIAL: Uncomment to log registry credentials
		// logrus.Tracef("Credentials: %v", registryAuth)
		r.Header.Add("Authorization", fmt.Sprintf("Basic %s", registryAuth))
	} else {
		logrus.Debug("No credentials found.")
	}

	var authResponse *http.Response
	if authResponse, err = helpers.HTTPClient().Do(r); err != nil {
		return "", err
	}
	defer func() { _ = authResponse.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(authResponse.Body, maxTokenResponseSize))
	if err != nil {
		return "", err
	}
	if authResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request to %s failed with %s", authURL.Host, authResponse.Status)
	}

	tokenResponse := &types.TokenResponse{}
	if err = json.Unmarshal(body, tokenResponse); err != nil {
		return "", fmt.Errorf("failed to parse the token response from %s: %w", authURL.Host, err)
	}

	token := tokenResponse.Token
	if token == "" {
		token = tokenResponse.AccessToken
	}
	if token == "" {
		return "", fmt.Errorf("the token response from %s did not include a token", authURL.Host)
	}

	return fmt.Sprintf("Bearer %s", token), nil
}

// GetAuthURL from the instructions in the challenge
func GetAuthURL(challenge string, imageRef ref.Named) (*url.URL, error) {
	_, values := parseChallenge(challenge)

	logrus.WithFields(logrus.Fields{
		"realm":   values["realm"],
		"service": values["service"],
	}).Debug("Checking challenge header content")
	if values["realm"] == "" {
		return nil, fmt.Errorf("challenge header did not include a realm to request a token from")
	}

	authURL, err := url.Parse(values["realm"])
	if err != nil || authURL.Host == "" {
		return nil, fmt.Errorf("challenge header included an invalid realm %q", values["realm"])
	}
	q := authURL.Query()
	// The service is optional, see https://distribution.github.io/distribution/spec/auth/token/
	if service := values["service"]; service != "" {
		q.Add("service", service)
	}

	scopeImage := ref.Path(imageRef)

	scope := fmt.Sprintf("repository:%s:pull", scopeImage)
	logrus.WithFields(logrus.Fields{"scope": scope, "image": imageRef.Name()}).Debug("Setting scope for auth token")
	q.Add("scope", scope)

	authURL.RawQuery = q.Encode()
	return authURL, nil
}

// GetChallengeURL returns the URL to check auth requirements
// for access to a given image
func GetChallengeURL(imageRef ref.Named) url.URL {
	host, _ := helpers.GetRegistryAddress(imageRef.Name())

	URL := url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/v2/",
	}
	return URL
}

// parseChallenge splits a WWW-Authenticate header into its lower-cased scheme and its parameters.
// Parameter names are lower-cased, while their values are kept as is, as realms are URLs.
func parseChallenge(header string) (scheme string, params map[string]string) {
	scheme, rest, _ := strings.Cut(strings.TrimSpace(header), " ")
	params = map[string]string{}

	for {
		rest = strings.TrimLeft(rest, " ,")
		separator := strings.IndexAny(rest, "=,")
		if separator < 0 {
			break
		}
		if rest[separator] == ',' {
			// a parameter without a value
			rest = rest[separator+1:]
			continue
		}

		key := strings.ToLower(strings.TrimSpace(rest[:separator]))
		rest = strings.TrimLeft(rest[separator+1:], " ")

		var value strings.Builder
		if strings.HasPrefix(rest, `"`) {
			// a quoted string, which may contain commas and escaped characters
			i := 1
			for ; i < len(rest) && rest[i] != '"'; i++ {
				if rest[i] == '\\' && i+1 < len(rest) {
					i++
				}
				value.WriteByte(rest[i])
			}
			rest = rest[min(i+1, len(rest)):]
		} else {
			end := strings.IndexByte(rest, ',')
			if end < 0 {
				end = len(rest)
			}
			value.WriteString(strings.TrimSpace(rest[:end]))
			rest = rest[end:]
		}
		params[key] = value.String()
	}

	return strings.ToLower(scheme), params
}
