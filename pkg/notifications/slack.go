package notifications

import (
	"fmt"
	"net/url"
	"strings"

	t "github.com/dallergy/watchtower/pkg/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	slackType = "slack"
)

type slackTypeNotifier struct {
	HookURL   string
	Username  string
	Channel   string
	IconEmoji string
	IconURL   string
}

func newSlackNotifier(c *cobra.Command) t.ConvertibleNotifier {
	flags := c.Flags()

	hookURL, _ := flags.GetString("notification-slack-hook-url")
	userName, _ := flags.GetString("notification-slack-identifier")
	channel, _ := flags.GetString("notification-slack-channel")
	emoji, _ := flags.GetString("notification-slack-icon-emoji")
	iconURL, _ := flags.GetString("notification-slack-icon-url")

	n := &slackTypeNotifier{
		HookURL:   hookURL,
		Username:  userName,
		Channel:   channel,
		IconEmoji: emoji,
		IconURL:   iconURL,
	}
	return n
}

func (s *slackTypeNotifier) GetURL(c *cobra.Command) (string, error) {
	trimmedURL := strings.TrimRight(s.HookURL, "/")
	trimmedURL = strings.TrimPrefix(trimmedURL, "https://")
	parts := strings.Split(trimmedURL, "/")

	if parts[0] == "discord.com" || parts[0] == "discordapp.com" {
		log.Debug("Detected a discord slack wrapper URL, using the discord service")
		if len(parts) < 4 {
			return "", fmt.Errorf("invalid discord webhook URL")
		}
		webhookID := parts[len(parts)-3]
		token := parts[len(parts)-2]

		botname := s.Username
		if botname == "" {
			botname = "watchtower"
		}

		discordURL := fmt.Sprintf("discord://%s@%s/%s", url.PathEscape(botname), webhookID, token)
		if s.IconURL != "" {
			discordURL += "?" + url.Values{"avatar_url": {s.IconURL}}.Encode()
		}
		return discordURL, nil
	}

	webhookToken := strings.Replace(s.HookURL, "https://hooks.slack.com/services/", "", 1)
	tokenParts := strings.Split(webhookToken, "/")
	if len(tokenParts) != 3 {
		return "", fmt.Errorf("invalid slack webhook URL")
	}

	botname := s.Username
	if botname == "" {
		botname = "watchtower"
	}

	slackURL := fmt.Sprintf("slack://%s@%s/%s/%s", url.PathEscape(botname), tokenParts[0], tokenParts[1], tokenParts[2])

	if s.Channel != "" {
		channel := s.Channel
		if !strings.HasPrefix(channel, "#") && !strings.HasPrefix(channel, "@") {
			channel = "#" + channel
		}
		slackURL += "/" + channel
	}

	q := url.Values{}
	if s.IconURL != "" {
		q.Set("icon_url", s.IconURL)
	} else if s.IconEmoji != "" {
		q.Set("icon_emoji", s.IconEmoji)
	}
	if encoded := q.Encode(); encoded != "" {
		slackURL += "?" + encoded
	}

	return slackURL, nil
}
