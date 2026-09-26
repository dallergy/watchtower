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
	msTeamsType = "msteams"
)

type msTeamsTypeNotifier struct {
	webHookURL string
	data       bool
}

func newMsTeamsNotifier(cmd *cobra.Command) t.ConvertibleNotifier {

	flags := cmd.Flags()

	webHookURL, _ := flags.GetString("notification-msteams-hook")
	if len(webHookURL) <= 0 {
		log.Fatal("Required argument --notification-msteams-hook(cli) or WATCHTOWER_NOTIFICATION_MSTEAMS_HOOK_URL(env) is empty.")
	}

	withData, _ := flags.GetBool("notification-msteams-data")
	n := &msTeamsTypeNotifier{
		webHookURL: webHookURL,
		data:       withData,
	}

	return n
}

func (n *msTeamsTypeNotifier) GetURL(c *cobra.Command) (string, error) {
	webhookURL, err := url.Parse(n.webHookURL)
	if err != nil {
		return "", err
	}

	path := strings.Trim(webhookURL.Path, "/")
	parts := strings.Split(path, "/")

	switch {
	case strings.Contains(path, "workflows/") && strings.Contains(path, "/triggers/"):
		// Power Automate workflows, which replace Office 365 connectors, are posted to as-is
		return n.webHookURL, nil
	case strings.HasSuffix(webhookURL.Host, ".webhook.office.com") && len(parts) >= 5 && parts[2] == "IncomingWebhook":
		team := strings.TrimSuffix(webhookURL.Host, ".webhook.office.com")
		tokens := append([]string{team, parts[1]}, parts[3:]...)
		return "msteams://" + strings.Join(tokens, "/"), nil
	case len(parts) >= 5 && parts[2] == "IncomingWebhook":
		return fmt.Sprintf("msteams://%s/%s/%s/", parts[1], parts[3], parts[4]), nil
	}

	return "", fmt.Errorf("invalid msteams webhook URL")
}
