package notifications

import (
	"fmt"
	"net/url"
	"strings"

	t "github.com/containrrr/watchtower/pkg/types"
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
		log.Debug("Detected a discord slack wrapper URL, using apprise discord service")
		webhookID := parts[len(parts)-3]
		token := parts[len(parts)-2]

		q := url.Values{}
		if s.Username != "" {
			q.Set("username", s.Username)
		} else {
			q.Set("username", "watchtower")
		}
		if s.IconURL != "" {
			q.Set("avatar", s.IconURL)
		}

		appriseURL := fmt.Sprintf("discord://%s/%s", webhookID, token)
		if encoded := q.Encode(); encoded != "" {
			appriseURL += "?" + encoded
		}
		return appriseURL, nil
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

	appriseURL := fmt.Sprintf("slack://%s@%s/%s/%s", botname, tokenParts[0], tokenParts[1], tokenParts[2])

	if s.Channel != "" {
		channel := s.Channel
		if !strings.HasPrefix(channel, "#") {
			channel = "#" + channel
		}
		appriseURL += "/" + channel
	}

	q := url.Values{}
	if s.IconURL != "" {
		q.Set("image", s.IconURL)
	} else if s.IconEmoji != "" {
		q.Set("image", s.IconEmoji)
	}
	if encoded := q.Encode(); encoded != "" {
		appriseURL += "?" + encoded
	}

	return appriseURL, nil
}
