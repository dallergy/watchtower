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
	if len(parts) < 5 || parts[2] != "IncomingWebhook" {
		return "", fmt.Errorf("invalid msteams webhook URL")
	}

	tokenA := parts[1]
	tokenB := parts[3]
	tokenC := parts[4]

	return fmt.Sprintf("msteams://%s/%s/%s/", tokenA, tokenB, tokenC), nil
}
