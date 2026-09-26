package apprise

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// jsonWebhook posts an Apprise JSON payload to any web server.
//
//	json://[user:password@]hostname[:port][/path]     (HTTP)
//	jsons://[user:password@]hostname[:port][/path]    (HTTPS)
//
// Parameters: method (POST or PUT), +Header=value to add HTTP headers
type jsonWebhook struct {
	web      *webClient
	endpoint string
	method   string
	headers  map[string]string
}

func newJSON(u *serviceURL, opts Options) (Service, error) {
	base, err := u.baseURL(u.scheme == "jsons")
	if err != nil {
		return nil, err
	}
	endpoint := base.JoinPath(u.path...)

	method := strings.ToUpper(u.param("method"))
	switch method {
	case "":
		method = http.MethodPost
	case http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		return nil, fmt.Errorf("unsupported method %q", method)
	}

	headers := map[string]string{}
	for key, value := range u.headers {
		headers[key] = value
	}
	if u.user != "" {
		headers["Authorization"] = basicAuth(u.user, u.password)
	}

	return &jsonWebhook{
		web:      newWebClient(u, opts),
		endpoint: endpoint.String(),
		method:   method,
		headers:  headers,
	}, nil
}

func (j *jsonWebhook) Scheme() string { return "json" }

func (j *jsonWebhook) Send(ctx context.Context, msg Message) error {
	payload := map[string]any{
		"version":     "1.0",
		"title":       msg.Title,
		"message":     msg.Body,
		"type":        "info",
		"attachments": []any{},
	}
	_, err := j.web.sendJSON(ctx, j.method, j.endpoint, payload, j.headers)
	return err
}
