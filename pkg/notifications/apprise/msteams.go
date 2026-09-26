package apprise

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// teams posts message cards using a Microsoft Teams incoming webhook (Office 365 connector).
//
//	msteams://team/tokenA/tokenB/tokenC[/tokenD]
//	msteams://tokenA/tokenB/tokenC    (legacy outlook.office.com webhooks)
//	https://team.webhook.office.com/webhookb2/...
type teams struct {
	web      *webClient
	endpoint string
}

// workflows posts adaptive cards to a Microsoft Teams (Power Automate) workflow, the replacement
// for Office 365 connectors.
//
//	workflows://hostname[:port]/workflow/signature
//	https://.../workflows/.../triggers/manual/paths/invoke?...
type workflows struct {
	web      *webClient
	endpoint string
}

const teamsThemeColor = "406170"

func newTeams(u *serviceURL, opts Options) (Service, error) {
	first := u.host
	if u.hasUserInfo {
		// tokenA of legacy webhooks contains an '@', which is parsed as user info
		first = u.user + "@" + u.host
	}
	tokens := append([]string{first}, u.path...)

	var endpoint string
	switch {
	case strings.Contains(first, "@") && len(tokens) == 3:
		endpoint = "https://outlook.office.com/webhook/" + strings.Join(escapeAll(teamsWebhookPath(tokens)), "/")
	case !strings.Contains(first, "@") && (len(tokens) == 4 || len(tokens) == 5):
		endpoint = fmt.Sprintf("https://%s.webhook.office.com/webhookb2/%s",
			url.PathEscape(first), strings.Join(escapeAll(teamsWebhookPath(tokens[1:])), "/"))
	default:
		return nil, fmt.Errorf("expected msteams://team/tokenA/tokenB/tokenC/tokenD")
	}

	return newTeamsWebhook(u, endpoint, opts), nil
}

// teamsWebhookPath inserts the IncomingWebhook segment after the first token
func teamsWebhookPath(tokens []string) []string {
	return append([]string{tokens[0], "IncomingWebhook"}, tokens[1:]...)
}

func newTeamsWebhook(u *serviceURL, endpoint string, opts Options) *teams {
	return &teams{web: newWebClient(u, opts), endpoint: endpoint}
}

func (t *teams) Scheme() string { return "msteams" }

func (t *teams) Send(ctx context.Context, msg Message) error {
	summary := msg.Title
	if summary == "" {
		summary = truncate(msg.Body, 80)
	}
	payload := map[string]any{
		"@type":      "MessageCard",
		"@context":   "https://schema.org/extensions",
		"summary":    summary,
		"themeColor": teamsThemeColor,
		"text":       strings.ReplaceAll(msg.Body, "\n", "<br>"),
	}
	if msg.Title != "" {
		payload["title"] = msg.Title
	}

	_, err := t.web.postJSON(ctx, t.endpoint, payload, nil)
	return err
}

func newWorkflows(u *serviceURL, opts Options) (Service, error) {
	if len(u.path) != 2 {
		return nil, fmt.Errorf("expected workflows://hostname/workflow/signature")
	}
	base, err := u.baseURL(true)
	if err != nil {
		return nil, err
	}

	endpoint := base.JoinPath("workflows", u.path[0], "triggers", "manual", "paths", "invoke")
	endpoint.RawQuery = url.Values{
		"api-version": {"2016-06-01"},
		"sp":          {"/triggers/manual/run"},
		"sv":          {"1.0"},
		"sig":         {u.path[1]},
	}.Encode()

	return newWorkflowsWebhook(u, endpoint.String(), opts), nil
}

func newWorkflowsWebhook(u *serviceURL, endpoint string, opts Options) *workflows {
	return &workflows{web: newWebClient(u, opts), endpoint: endpoint}
}

func (w *workflows) Scheme() string { return "workflows" }

func (w *workflows) Send(ctx context.Context, msg Message) error {
	var blocks []any
	if msg.Title != "" {
		blocks = append(blocks, map[string]any{
			"type":   "TextBlock",
			"text":   msg.Title,
			"weight": "Bolder",
			"size":   "Medium",
			"wrap":   true,
		})
	}
	blocks = append(blocks, map[string]any{
		"type": "TextBlock",
		"text": msg.Body,
		"wrap": true,
	})

	payload := map[string]any{
		"type": "message",
		"attachments": []any{
			map[string]any{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]any{
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"type":    "AdaptiveCard",
					"version": "1.4",
					"body":    blocks,
					"msteams": map[string]any{"width": "Full"},
				},
			},
		},
	}

	_, err := w.web.postJSON(ctx, w.endpoint, payload, nil)
	return err
}
