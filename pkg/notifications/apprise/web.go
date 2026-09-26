package apprise

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dallergy/watchtower/internal/meta"
)

const (
	requestTimeout = 30 * time.Second
	// maxRetryAfter is the longest rate limit back-off that is waited out before retrying once
	maxRetryAfter = 10 * time.Second
	// maxResponseSize is the largest amount of a response body that is read
	maxResponseSize = 64 * 1024
)

// webClient sends HTTP requests on behalf of a service
type webClient struct {
	client *http.Client
}

func newWebClient(u *serviceURL, opts Options) *webClient {
	transport := opts.Transport
	if transport == nil {
		defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
		if !u.boolParam("verify", true) {
			defaultTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // explicitly requested using verify=no
		}
		transport = defaultTransport
	}

	return &webClient{
		client: &http.Client{
			Timeout:   requestTimeout,
			Transport: transport,
		},
	}
}

// postJSON sends the payload as JSON and returns the response body
func (w *webClient) postJSON(ctx context.Context, endpoint string, payload any, headers map[string]string) ([]byte, error) {
	return w.sendJSON(ctx, http.MethodPost, endpoint, payload, headers)
}

// sendJSON sends the payload as JSON using the given method and returns the response body
func (w *webClient) sendJSON(ctx context.Context, method string, endpoint string, payload any, headers map[string]string) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode payload: %w", err)
	}

	allHeaders := map[string]string{"Content-Type": "application/json"}
	for key, value := range headers {
		allHeaders[key] = value
	}
	return w.do(ctx, method, endpoint, body, allHeaders)
}

// postForm sends the values as a form and returns the response body
func (w *webClient) postForm(ctx context.Context, endpoint string, values url.Values) ([]byte, error) {
	headers := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	return w.do(ctx, http.MethodPost, endpoint, []byte(values.Encode()), headers)
}

// do sends a request, retrying once when rate limited, and returns the body of a successful response
func (w *webClient) do(ctx context.Context, method string, endpoint string, body []byte, headers map[string]string) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, sanitizeError(err)
		}
		req.Header.Set("User-Agent", meta.UserAgent)
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		res, err := w.client.Do(req)
		if err != nil {
			return nil, sanitizeError(err)
		}
		resBody, readErr := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
		_ = res.Body.Close()

		if res.StatusCode == http.StatusTooManyRequests && attempt == 0 {
			if delay, ok := retryAfter(res.Header.Get("Retry-After")); ok {
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
		}

		if res.StatusCode < 200 || res.StatusCode > 299 {
			return nil, fmt.Errorf("%s responded with %s%s", req.URL.Host, res.Status, describeBody(resBody))
		}
		if readErr != nil {
			return nil, fmt.Errorf("failed to read response from %s: %w", req.URL.Host, sanitizeError(readErr))
		}
		return resBody, nil
	}
}

func retryAfter(header string) (time.Duration, bool) {
	seconds, err := strconv.ParseFloat(strings.TrimSpace(header), 64)
	if err != nil || seconds < 0 {
		return 0, false
	}
	delay := time.Duration(seconds * float64(time.Second))
	return delay, delay <= maxRetryAfter
}

// describeBody returns a short excerpt of an error response, for use in error messages
func describeBody(body []byte) string {
	text := strings.Join(strings.Fields(string(body)), " ")
	if text == "" {
		return ""
	}
	const maxLength = 200
	if len(text) > maxLength {
		text = text[:maxLength] + "…"
	}
	return ": " + text
}

// sanitizeError strips the request URL from HTTP client errors, as it can contain secret tokens
func sanitizeError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		host := "service"
		if parsed, parseErr := url.Parse(urlErr.URL); parseErr == nil && parsed.Host != "" {
			host = parsed.Host
		}
		return fmt.Errorf("%s %s: %w", urlErr.Op, host, urlErr.Err)
	}
	return err
}

// basicAuth returns the value of an Authorization header for HTTP basic authentication
func basicAuth(user string, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
}

// truncate shortens text to at most limit runes, marking it as truncated
func truncate(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit-1]) + "…"
}

// joinTitle prefixes the body with the title on its own line, for services without a title field
func joinTitle(msg Message) string {
	if msg.Title == "" {
		return msg.Body
	}
	if msg.Body == "" {
		return msg.Title
	}
	return msg.Title + "\n" + msg.Body
}
