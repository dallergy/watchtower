package apprise

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// telegram sends messages through a Telegram bot.
//
//	tgram://bot_token/chat_id[/chat_id...]
//
// Chat IDs are numeric IDs or @channel names. Parameters: silent (yes/no), topic (message thread ID)
type telegram struct {
	web      *webClient
	endpoint string
	chatIDs  []string
	silent   bool
	topic    int
}

const (
	telegramAPIURL = "https://api.telegram.org"
	// telegramTextLimit is the maximum length of a message imposed by Telegram
	telegramTextLimit = 4096
)

func newTelegram(u *serviceURL, opts Options) (Service, error) {
	token := strings.TrimPrefix(u.host, "bot")
	if !strings.Contains(token, ":") {
		return nil, fmt.Errorf("expected tgram://bot_token/chat_id, where the bot token looks like 123456789:AbCdEf")
	}
	if len(u.path) == 0 {
		return nil, fmt.Errorf("missing chat ID")
	}

	topic := 0
	if rawTopic := u.param("topic", "thread"); rawTopic != "" {
		var err error
		if topic, err = strconv.Atoi(rawTopic); err != nil {
			return nil, fmt.Errorf("invalid topic %q", rawTopic)
		}
	}

	return &telegram{
		web:      newWebClient(u, opts),
		endpoint: telegramAPIURL + "/bot" + token + "/sendMessage",
		chatIDs:  u.path,
		silent:   u.boolParam("silent", false),
		topic:    topic,
	}, nil
}

func (t *telegram) Scheme() string { return "tgram" }

func (t *telegram) Send(ctx context.Context, msg Message) error {
	text := truncate(joinTitle(msg), telegramTextLimit)

	for _, chatID := range t.chatIDs {
		payload := map[string]any{
			"chat_id":              chatID,
			"text":                 text,
			"disable_notification": t.silent,
		}
		if t.topic > 0 {
			payload["message_thread_id"] = t.topic
		}

		body, err := t.web.postJSON(ctx, t.endpoint, payload, nil)
		if err != nil {
			return fmt.Errorf("chat %s: %w", chatID, err)
		}

		var result struct {
			OK          bool   `json:"ok"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(body, &result); err == nil && !result.OK {
			return fmt.Errorf("chat %s: telegram API error: %s", chatID, result.Description)
		}
	}
	return nil
}
