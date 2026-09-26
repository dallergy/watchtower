package apprise

import (
	"context"
	"fmt"
	"net/url"
)

// ntfy publishes messages to ntfy topics.
//
//	ntfy://topic                                      (the public https://ntfy.sh server)
//	ntfy://[user:password@]hostname[:port]/topic      (HTTP)
//	ntfys://[user:password@]hostname[:port]/topic     (HTTPS)
//
// Parameters: priority (min, low, default, high, max or 1-5), tags, click, token (access token)
type ntfy struct {
	web      *webClient
	endpoint string
	topics   []string
	priority int
	tags     []string
	click    string
	headers  map[string]string
}

var ntfyPriorities = map[string]int{
	"min":     1,
	"low":     2,
	"default": 3,
	"normal":  3,
	"high":    4,
	"max":     5,
	"urgent":  5,
}

const ntfyCloudURL = "https://ntfy.sh"

func newNtfy(u *serviceURL, opts Options) (Service, error) {
	endpoint := ntfyCloudURL
	topics := u.path
	if len(topics) == 0 {
		// cloud mode, where the host is the topic
		if u.host == "" {
			return nil, fmt.Errorf("missing topic")
		}
		topics = []string{u.host}
	} else {
		base, err := u.baseURL(u.scheme == "ntfys")
		if err != nil {
			return nil, err
		}
		endpoint = base.String()
	}

	priority, err := parsePriority(u.param("priority"), ntfyPriorities, 0, 1, 5)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{}
	if token := u.param("token", "auth"); token != "" {
		headers["Authorization"] = "Bearer " + token
	} else if u.user != "" {
		headers["Authorization"] = basicAuth(u.user, u.password)
	}

	return &ntfy{
		web:      newWebClient(u, opts),
		endpoint: endpoint,
		topics:   topics,
		priority: priority,
		tags:     u.listParam("tags", "tag"),
		click:    u.param("click"),
		headers:  headers,
	}, nil
}

func (n *ntfy) Scheme() string { return "ntfy" }

func (n *ntfy) Send(ctx context.Context, msg Message) error {
	for _, topic := range n.topics {
		payload := map[string]any{
			"topic":   topic,
			"message": msg.Body,
		}
		if msg.Title != "" {
			payload["title"] = msg.Title
		}
		if n.priority > 0 {
			payload["priority"] = n.priority
		}
		if len(n.tags) > 0 {
			payload["tags"] = n.tags
		}
		if n.click != "" {
			payload["click"] = n.click
		}

		// Publishing as JSON to the root URL avoids having to encode the title as a header
		if _, err := n.web.postJSON(ctx, n.endpoint, payload, n.headers); err != nil {
			return fmt.Errorf("topic %s: %w", url.PathEscape(topic), err)
		}
	}
	return nil
}
