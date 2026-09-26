package apprise

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// gotify sends messages to a Gotify server.
//
//	gotify://hostname[:port][/path]/token     (HTTP)
//	gotifys://hostname[:port][/path]/token    (HTTPS)
//
// Parameters: priority (low, moderate, normal, high, emergency or 0-10), format (text or markdown)
type gotify struct {
	web      *webClient
	endpoint string
	token    string
	priority int
	markdown bool
}

var gotifyPriorities = map[string]int{
	"low":       1,
	"moderate":  3,
	"normal":    5,
	"high":      8,
	"emergency": 10,
}

func newGotify(u *serviceURL, opts Options) (Service, error) {
	if len(u.path) == 0 {
		return nil, fmt.Errorf("missing application token")
	}

	base, err := u.baseURL(u.scheme == "gotifys")
	if err != nil {
		return nil, err
	}
	token := u.path[len(u.path)-1]
	prefix := u.path[:len(u.path)-1]
	base = base.JoinPath(append(prefix, "message")...)

	priority, err := parsePriority(u.param("priority"), gotifyPriorities, gotifyPriorities["normal"], 0, 10)
	if err != nil {
		return nil, err
	}

	return &gotify{
		web:      newWebClient(u, opts),
		endpoint: base.String(),
		token:    token,
		priority: priority,
		markdown: strings.EqualFold(u.param("format"), "markdown"),
	}, nil
}

func (g *gotify) Scheme() string { return "gotify" }

func (g *gotify) Send(ctx context.Context, msg Message) error {
	payload := map[string]any{
		"message":  msg.Body,
		"priority": g.priority,
	}
	if msg.Title != "" {
		payload["title"] = msg.Title
	}
	if g.markdown {
		payload["extras"] = map[string]any{
			"client::display": map[string]string{"contentType": "text/markdown"},
		}
	}

	_, err := g.web.postJSON(ctx, g.endpoint, payload, map[string]string{"X-Gotify-Key": g.token})
	return err
}

// parsePriority resolves a named or numeric priority, ensuring numeric values are within bounds
func parsePriority(value string, names map[string]int, fallback int, lowest int, highest int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	if priority, found := names[strings.ToLower(value)]; found {
		return priority, nil
	}
	priority, err := strconv.Atoi(value)
	if err != nil || priority < lowest || priority > highest {
		return 0, fmt.Errorf("invalid priority %q", value)
	}
	return priority, nil
}
