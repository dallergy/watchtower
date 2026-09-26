package apprise

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// pushover sends messages using Pushover.
//
//	pover://user_key@app_token[/device...]
//
// Parameters: priority (lowest, low, normal, high, emergency or -2 to 2), sound
type pushover struct {
	web      *webClient
	user     string
	token    string
	devices  []string
	priority int
	sound    string
}

const (
	pushoverAPIURL = "https://api.pushover.net/1/messages.json"
	// pushoverTitleLimit and pushoverMessageLimit are the length limits imposed by Pushover
	pushoverTitleLimit   = 250
	pushoverMessageLimit = 1024
	// emergency messages are repeated until acknowledged, using these intervals (in seconds)
	pushoverRetry  = 60
	pushoverExpire = 3600
)

var pushoverPriorities = map[string]int{
	"lowest":    -2,
	"low":       -1,
	"normal":    0,
	"high":      1,
	"emergency": 2,
}

func newPushover(u *serviceURL, opts Options) (Service, error) {
	if u.user == "" || u.host == "" {
		return nil, fmt.Errorf("expected pover://user_key@app_token")
	}

	priority, err := parsePriority(u.param("priority"), pushoverPriorities, 0, -2, 2)
	if err != nil {
		return nil, err
	}

	return &pushover{
		web:      newWebClient(u, opts),
		user:     u.user,
		token:    u.host,
		devices:  u.path,
		priority: priority,
		sound:    u.param("sound"),
	}, nil
}

func (p *pushover) Scheme() string { return "pover" }

func (p *pushover) Send(ctx context.Context, msg Message) error {
	values := url.Values{
		"token":    {p.token},
		"user":     {p.user},
		"message":  {truncate(msg.Body, pushoverMessageLimit)},
		"priority": {strconv.Itoa(p.priority)},
	}
	if msg.Title != "" {
		values.Set("title", truncate(msg.Title, pushoverTitleLimit))
	}
	if len(p.devices) > 0 {
		values.Set("device", strings.Join(p.devices, ","))
	}
	if p.sound != "" {
		values.Set("sound", p.sound)
	}
	if p.priority == pushoverPriorities["emergency"] {
		values.Set("retry", strconv.Itoa(pushoverRetry))
		values.Set("expire", strconv.Itoa(pushoverExpire))
	}

	body, err := p.web.postForm(ctx, pushoverAPIURL, values)
	if err != nil {
		return err
	}

	var result struct {
		Status int      `json:"status"`
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal(body, &result); err == nil && result.Status != 1 {
		return fmt.Errorf("pushover API error: %s", strings.Join(result.Errors, ", "))
	}
	return nil
}
