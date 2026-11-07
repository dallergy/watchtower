package apprise

import (
	"testing"
)

// FuzzParse ensures that no notification URL, however malformed, makes the parser panic
func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"gotifys://gotify.example.com/sub/token?priority=high",
		"ntfy://topic",
		"ntfys://user:pass@[::1]:8443/a/b?tags=x,y&priority=5",
		"slack://bot@T/B/C/#chan/@user/+ID?icon_emoji=whale",
		"slack://xoxb-1/#c",
		"discord://bot@1/token?thread=2",
		"tgram://123:abc/-100/@chan?topic=7",
		"pover://user@token/dev?priority=-2",
		"msteams://aaa@bbb/ccc/ddd/",
		"msteams://team/a/b/c/d",
		"workflows://host:443/wf/sig",
		"mailtos://u%40x:p@smtp.example.com:465/to@example.com?cc=a@b.c&mode=ssl",
		"jsons://u:p@host:1/p?method=put&+X-Key=v",
		"https://hooks.slack.com/services/A/B/C",
		"https://discord.com/api/webhooks/1/t",
		"https://x.webhook.office.com/webhookb2/a/IncomingWebhook/b/c",
		"https://h/workflows/a/triggers/manual/paths/invoke?sig=s",
		"",
		"://",
		"gotify://",
		"gotify://%zz/t",
		"mailto://@",
		"tgram://bot:/x",
		"msteams://@/",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, rawURL string) {
		_, _ = Parse(rawURL, Options{})
		_ = IsSupported(rawURL)
		_ = Scheme(rawURL)
	})
}
