package apprise

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// discord posts messages using a Discord webhook.
//
//	discord://[botname@]webhook_id/webhook_token
//	https://discord.com/api/webhooks/webhook_id/webhook_token
//
// Parameters: avatar_url, tts (yes/no), thread (thread ID)
type discord struct {
	web       *webClient
	endpoint  string
	username  string
	avatarURL string
	tts       bool
}

const (
	discordWebhookURL = "https://discord.com/api/webhooks"
	// discordColor is the accent color used for message embeds
	discordColor = 0x406170
	// discordTitleLimit and discordDescriptionLimit are the embed length limits imposed by Discord
	discordTitleLimit       = 256
	discordDescriptionLimit = 4096
)

func newDiscord(u *serviceURL, opts Options) (Service, error) {
	if u.host == "" || len(u.path) == 0 {
		return nil, fmt.Errorf("expected discord://webhook_id/webhook_token")
	}
	return newDiscordWebhook(u, u.host, u.path[0], opts), nil
}

func newDiscordWebhook(u *serviceURL, webhookID string, token string, opts Options) *discord {
	endpoint, _ := url.Parse(discordWebhookURL)
	endpoint = endpoint.JoinPath(webhookID, token)
	query := url.Values{"wait": {"true"}}
	if thread := u.param("thread"); thread != "" {
		query.Set("thread_id", thread)
	}
	endpoint.RawQuery = query.Encode()

	d := &discord{
		web:       newWebClient(u, opts),
		endpoint:  endpoint.String(),
		username:  u.user,
		avatarURL: u.param("avatar_url"),
		tts:       u.boolParam("tts", false),
	}
	// Earlier releases of this fork used username and avatar parameters
	if d.username == "" {
		d.username = u.param("username", "botname")
	}
	if avatar := u.param("avatar"); d.avatarURL == "" && strings.HasPrefix(avatar, "http") {
		d.avatarURL = avatar
	}
	return d
}

func (d *discord) Scheme() string { return "discord" }

func (d *discord) Send(ctx context.Context, msg Message) error {
	embed := map[string]any{
		"description": truncate(msg.Body, discordDescriptionLimit),
		"color":       discordColor,
	}
	if msg.Title != "" {
		embed["title"] = truncate(msg.Title, discordTitleLimit)
	}

	payload := map[string]any{
		"embeds": []any{embed},
		"tts":    d.tts,
	}
	if d.username != "" {
		payload["username"] = d.username
	}
	if d.avatarURL != "" {
		payload["avatar_url"] = d.avatarURL
	}

	_, err := d.web.postJSON(ctx, d.endpoint, payload, nil)
	return err
}
