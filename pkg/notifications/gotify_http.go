package notifications

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type gotifyHTTPRouter struct {
	client    *http.Client
	endpoints []gotifyEndpoint
}

type gotifyEndpoint struct {
	messageURL string
	host       string
}

type gotifyPayload struct {
	Title    string `json:"title,omitempty"`
	Message  string `json:"message"`
	Priority int    `json:"priority"`
}

func newGotifyHTTPRouter(urls []string, skipVerify bool) (*gotifyHTTPRouter, error) {
	endpoints := make([]gotifyEndpoint, 0, len(urls))
	for _, raw := range urls {
		endpoint, err := parseGotifyAppriseURL(raw)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, endpoint)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if skipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	return &gotifyHTTPRouter{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		endpoints: endpoints,
	}, nil
}

func parseGotifyAppriseURL(raw string) (gotifyEndpoint, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return gotifyEndpoint{}, fmt.Errorf("invalid gotify URL: %w", err)
	}

	httpScheme := "https"
	switch u.Scheme {
	case "gotify":
		httpScheme = gotifyHTTPScheme(u.Hostname())
	case "gotifys":
		httpScheme = "https"
	default:
		return gotifyEndpoint{}, fmt.Errorf("unsupported gotify scheme %q", u.Scheme)
	}

	if u.Host == "" {
		return gotifyEndpoint{}, fmt.Errorf("gotify URL is missing a host")
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return gotifyEndpoint{}, fmt.Errorf("gotify URL is missing an application token")
	}

	token := parts[len(parts)-1]
	prefix := strings.Join(parts[:len(parts)-1], "/")

	messageURL := &url.URL{
		Scheme: httpScheme,
		Host:   u.Host,
		Path:   joinGotifyPath(prefix, "message"),
	}
	query := messageURL.Query()
	query.Set("token", token)
	messageURL.RawQuery = query.Encode()

	return gotifyEndpoint{
		messageURL: messageURL.String(),
		host:       u.Host,
	}, nil
}

func gotifyHTTPScheme(host string) string {
	if host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1" || net.ParseIP(host) != nil {
		return "http"
	}
	return "https"
}

func joinGotifyPath(prefix, name string) string {
	if prefix == "" {
		return "/" + name
	}
	return "/" + strings.Trim(prefix, "/") + "/" + name
}

func (r *gotifyHTTPRouter) Send(message string, params *notificationParams) []error {
	payload := gotifyPayload{
		Message:  message,
		Priority: 5,
	}
	if params != nil {
		if title, ok := params.Title(); ok {
			payload.Title = title
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return perURLErrors(endpointHosts(r.endpoints), fmt.Errorf("failed to marshal gotify payload: %w", err))
	}

	errs := make([]error, len(r.endpoints))
	for i, endpoint := range r.endpoints {
		errs[i] = r.post(endpoint, body)
	}
	return errs
}

func (r *gotifyHTTPRouter) post(endpoint gotifyEndpoint, body []byte) error {
	req, err := http.NewRequest(http.MethodPost, endpoint.messageURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create gotify request for %s: %w", endpoint.host, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send gotify notification to %s: %w", endpoint.host, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gotify at %s returned status %d", endpoint.host, resp.StatusCode)
	}
	return nil
}

func endpointHosts(endpoints []gotifyEndpoint) []string {
	hosts := make([]string, len(endpoints))
	for i, endpoint := range endpoints {
		hosts[i] = endpoint.host
	}
	return hosts
}

func splitGotifyURLs(urls []string) (gotify []string, other []string) {
	for _, u := range urls {
		switch GetScheme(u) {
		case "gotify", "gotifys":
			gotify = append(gotify, u)
		default:
			other = append(other, u)
		}
	}
	return gotify, other
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
	var children []router
	for _, r := range routers {
		if r != nil {
			children = append(children, r)
		}
	}
	switch len(children) {
	case 0:
		return &noopRouter{}
	case 1:
		return children[0]
	default:
		return &fanoutRouter{children: children}
	}
}
