package apprise

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// slack posts messages to Slack, using either an incoming webhook or a bot token.
//
//	slack://[botname@]tokenA/tokenB/tokenC[/#channel...]    (incoming webhook)
//	slack://[botname@]xoxb-bot-token/#channel[/#channel...]  (bot token)
//	https://hooks.slack.com/services/tokenA/tokenB/tokenC
//
// Channels are given as #name, @user or +ID. Parameters: icon_url, icon_emoji
type slack struct {
	web       *webClient
	webhook   string
	botToken  string
	channels  []string
	username  string
	iconURL   string
	iconEmoji string
}

const (
	slackWebhookURL     = "https://hooks.slack.com/services"
	slackPostMessageURL = "https://slack.com/api/chat.postMessage"
)

func newSlack(u *serviceURL, opts Options) (Service, error) {
	if strings.HasPrefix(u.host, "xox") {
		if len(u.path) == 0 {
			return nil, fmt.Errorf("a channel is required when using a bot token")
		}
		s := newSlackService(u, opts)
		s.botToken = u.host
		s.channels = slackChannels(u.path)
		return s, nil
	}

	if u.host == "" || len(u.path) < 2 {
		return nil, fmt.Errorf("expected slack://tokenA/tokenB/tokenC")
	}
	tokens := []string{u.host, u.path[0], u.path[1]}
	return newSlackWebhook(u, tokens, slackChannels(u.path[2:]), opts), nil
}

func newSlackWebhook(u *serviceURL, tokens []string, channels []string, opts Options) *slack {
	s := newSlackService(u, opts)
	s.webhook = slackWebhookURL + "/" + strings.Join(escapeAll(tokens), "/")
	s.channels = channels
	return s
}

func newSlackService(u *serviceURL, opts Options) *slack {
	s := &slack{
		web:       newWebClient(u, opts),
		username:  u.user,
		iconURL:   u.param("icon_url"),
		iconEmoji: u.param("icon_emoji"),
	}
	// Earlier releases of this fork converted the legacy icon flags to an image parameter
	if image := u.param("image"); image != "" && s.iconURL == "" && s.iconEmoji == "" {
		if strings.HasPrefix(image, "http://") || strings.HasPrefix(image, "https://") {
			s.iconURL = image
		} else if !isBoolWord(image) {
			s.iconEmoji = image
		}
	}
	if s.iconEmoji != "" && !strings.HasPrefix(s.iconEmoji, ":") {
		s.iconEmoji = ":" + strings.Trim(s.iconEmoji, ":") + ":"
	}
	return s
}

func slackChannels(segments []string) []string {
	channels := make([]string, 0, len(segments))
	for _, channel := range segments {
		switch {
		case strings.HasPrefix(channel, "+"):
			channels = append(channels, channel[1:])
		case strings.HasPrefix(channel, "#"), strings.HasPrefix(channel, "@"):
			channels = append(channels, channel)
		default:
			channels = append(channels, "#"+channel)
		}
	}
	return channels
}

func (s *slack) Scheme() string { return "slack" }

func (s *slack) Send(ctx context.Context, msg Message) error {
	text := msg.Body
	if msg.Title != "" {
		text = "*" + msg.Title + "*\n" + msg.Body
	}

	channels := s.channels
	if len(channels) == 0 {
		// post to the default channel of the webhook
		channels = []string{""}
	}

	for _, channel := range channels {
		payload := map[string]any{"text": text}
		if channel != "" {
			payload["channel"] = channel
		}
		if s.username != "" {
			payload["username"] = s.username
		}
		if s.iconURL != "" {
			payload["icon_url"] = s.iconURL
		} else if s.iconEmoji != "" {
			payload["icon_emoji"] = s.iconEmoji
		}

		if err := s.post(ctx, payload); err != nil {
			if channel != "" {
				return fmt.Errorf("channel %s: %w", channel, err)
			}
			return err
		}
	}
	return nil
}

func (s *slack) post(ctx context.Context, payload map[string]any) error {
	if s.botToken == "" {
		_, err := s.web.postJSON(ctx, s.webhook, payload, nil)
		return err
	}

	body, err := s.web.postJSON(ctx, slackPostMessageURL, payload, map[string]string{
		"Authorization": "Bearer " + s.botToken,
	})
	if err != nil {
		return err
	}

	// The Web API reports errors in the body of successful responses
	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to decode slack response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}
	return nil
}

func escapeAll(segments []string) []string {
	escaped := make([]string, len(segments))
	for i, segment := range segments {
		escaped[i] = url.PathEscape(segment)
	}
	return escaped
}

func isBoolWord(value string) bool {
	switch strings.ToLower(value) {
	case "yes", "no", "true", "false", "on", "off", "1", "0", "y", "n":
		return true
	}
	return false
}
