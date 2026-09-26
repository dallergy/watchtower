package apprise

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordedRequest is a request received by the test server
type recordedRequest struct {
	Method string
	URL    string
	Header http.Header
	Body   []byte
}

func (r recordedRequest) JSON(t *testing.T) map[string]any {
	t.Helper()
	var payload map[string]any
	require.NoError(t, json.Unmarshal(r.Body, &payload), string(r.Body))
	return payload
}

// testServer records requests and answers them using the configured responder
type testServer struct {
	*httptest.Server
	mu        sync.Mutex
	requests  []recordedRequest
	responder func(w http.ResponseWriter, attempt int)
}

func newTestServer(t *testing.T) *testServer {
	s := &testServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		// The transport redirects every request here, while leaving the Host header untouched
		s.requests = append(s.requests, recordedRequest{
			Method: r.Method,
			URL:    "https://" + r.Host + r.URL.RequestURI(),
			Header: r.Header.Clone(),
			Body:   body,
		})
		attempt := len(s.requests)
		responder := s.responder
		s.mu.Unlock()

		if responder != nil {
			responder(w, attempt)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"status":1}`))
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *testServer) options() Options {
	target, _ := url.Parse(s.URL)
	return Options{Transport: redirectTransport{target: target}}
}

func (s *testServer) only(t *testing.T) recordedRequest {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	require.Len(t, s.requests, 1)
	return s.requests[0]
}

func (s *testServer) all() []recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]recordedRequest(nil), s.requests...)
}

// redirectTransport sends all requests to the test server, keeping the original Host header
type redirectTransport struct {
	target *url.URL
}

func (rt redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	redirected := req.Clone(req.Context())
	redirected.Host = req.URL.Host
	redirected.URL.Scheme = rt.target.Scheme
	redirected.URL.Host = rt.target.Host
	return http.DefaultTransport.RoundTrip(redirected)
}

var testMessage = Message{Title: "Watchtower updates on host", Body: "Updated app (app:latest)\nFound 2 updates"}

func send(t *testing.T, rawURL string, server *testServer) {
	t.Helper()
	service, err := Parse(rawURL, server.options())
	require.NoError(t, err)
	require.NoError(t, service.Send(context.Background(), testMessage))
}

func TestParseServiceURL(t *testing.T) {
	u, err := parseServiceURL("TGRAM://123456789:AbC-dEf/%40channel/-100123?Silent=Yes&+X-Custom=a%20b&to=a+b")
	require.NoError(t, err)
	assert.Equal(t, "tgram", u.scheme)
	assert.Equal(t, "123456789:AbC-dEf", u.host)
	assert.Equal(t, []string{"@channel", "-100123"}, u.path)
	assert.Equal(t, "Yes", u.params["silent"])
	assert.Equal(t, "a b", u.headers["X-Custom"])
	assert.Equal(t, "a b", u.params["to"])

	u, err = parseServiceURL("slack://bot@T1/B2/C3/#general/@user")
	require.NoError(t, err)
	assert.Equal(t, "bot", u.user)
	assert.Equal(t, "T1", u.host)
	assert.Equal(t, []string{"B2", "C3", "#general", "@user"}, u.path)

	u, err = parseServiceURL("mailtos://user%40example.com:p%40ss:word@[::1]:2525/")
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", u.user)
	assert.Equal(t, "p@ss:word", u.password)
	hostname, port, err := u.hostPort()
	require.NoError(t, err)
	assert.Equal(t, "::1", hostname)
	assert.Equal(t, 2525, port)

	_, _, err = (&serviceURL{host: "example.com:notaport"}).hostPort()
	assert.Error(t, err)

	_, err = parseServiceURL("no-scheme")
	assert.Error(t, err)
}

func TestSchemes(t *testing.T) {
	schemes := Schemes()
	assert.Contains(t, schemes, "gotifys")
	assert.Contains(t, schemes, "mailtos")
	assert.IsIncreasing(t, schemes)
}

func TestUnsupported(t *testing.T) {
	_, err := Parse("matrix://user:pass@example.com/#room", Options{})
	var unsupported UnsupportedError
	require.ErrorAs(t, err, &unsupported)
	assert.Equal(t, "matrix", unsupported.Scheme)
	assert.False(t, IsSupported("matrix://example.com"))
	assert.False(t, IsSupported("https://example.com/webhook"))
	assert.True(t, IsSupported("https://discord.com/api/webhooks/1/token"))
	assert.True(t, IsSupported("GOTIFYS://example.com/token"))

	_, err = Parse("https://example.com/webhook", Options{})
	assert.ErrorAs(t, err, &unsupported)
}

func TestInvalidURLs(t *testing.T) {
	for _, rawURL := range []string{
		"gotify://example.com",
		"gotify://example.com/token?priority=11",
		"gotify://example.com:99999/token",
		"slack://T1/B2",
		"slack://xoxb-123",
		"discord://123",
		"tgram://123456789/12345",
		"tgram://123456789:token",
		"pover://token",
		"pover://user@token?priority=5",
		"msteams://a/b",
		"workflows://host/only-workflow",
		"json://example.com?method=DELETE",
		"ntfy://",
		"mailto://example.com?mode=tls",
		"mailto://example.com?to=not-an-address",
	} {
		_, err := Parse(rawURL, Options{})
		assert.Error(t, err, rawURL)
	}
}

func TestGotify(t *testing.T) {
	server := newTestServer(t)
	send(t, "gotifys://gotify.example.com/sub/path/AbCdEf?priority=high&format=markdown", server)

	req := server.only(t)
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "https://gotify.example.com/sub/path/message", req.URL)
	assert.Equal(t, "AbCdEf", req.Header.Get("X-Gotify-Key"))
	assert.NotContains(t, req.URL, "AbCdEf", "the token should not be sent in the URL")

	payload := req.JSON(t)
	assert.Equal(t, testMessage.Title, payload["title"])
	assert.Equal(t, testMessage.Body, payload["message"])
	assert.EqualValues(t, 8, payload["priority"])
	assert.Equal(t, "text/markdown", payload["extras"].(map[string]any)["client::display"].(map[string]any)["contentType"])
}

func TestGotifySchemes(t *testing.T) {
	for rawURL, expected := range map[string]string{
		"gotify://gotify:8080/token":  "http://gotify:8080/message",
		"gotifys://gotify/token":      "https://gotify/message",
		"gotify://192.168.1.2/token":  "http://192.168.1.2/message",
		"gotify://[fd00::1]:80/token": "http://[fd00::1]:80/message",
	} {
		service, err := Parse(rawURL, Options{})
		require.NoError(t, err)
		assert.Equal(t, expected, service.(*gotify).endpoint, rawURL)
		assert.Equal(t, 5, service.(*gotify).priority)
	}
}

func TestNtfy(t *testing.T) {
	server := newTestServer(t)
	send(t, "ntfys://user:pass@ntfy.example.com/updates/alerts?priority=max&tags=whale,warning&click=https://example.com", server)

	requests := server.all()
	require.Len(t, requests, 2)
	for i, topic := range []string{"updates", "alerts"} {
		req := requests[i]
		assert.Equal(t, "https://ntfy.example.com/", req.URL)
		assert.Equal(t, basicAuth("user", "pass"), req.Header.Get("Authorization"))
		payload := req.JSON(t)
		assert.Equal(t, topic, payload["topic"])
		assert.Equal(t, testMessage.Title, payload["title"])
		assert.Equal(t, testMessage.Body, payload["message"])
		assert.EqualValues(t, 5, payload["priority"])
		assert.Equal(t, []any{"whale", "warning"}, payload["tags"])
		assert.Equal(t, "https://example.com", payload["click"])
	}
}

func TestNtfyCloudWithToken(t *testing.T) {
	server := newTestServer(t)
	send(t, "ntfy://my-topic?token=tk_secret", server)

	req := server.only(t)
	assert.Equal(t, "https://ntfy.sh/", req.URL)
	assert.Equal(t, "Bearer tk_secret", req.Header.Get("Authorization"))
	assert.Equal(t, "my-topic", req.JSON(t)["topic"])
	assert.NotContains(t, req.JSON(t), "priority")
}

func TestSlackWebhook(t *testing.T) {
	server := newTestServer(t)
	send(t, "slack://watchtower@T000/B000/XXXX/#updates/@admin/+C0123?icon_emoji=whale", server)

	requests := server.all()
	require.Len(t, requests, 3)
	for i, channel := range []string{"#updates", "@admin", "C0123"} {
		req := requests[i]
		assert.Equal(t, "https://hooks.slack.com/services/T000/B000/XXXX", req.URL)
		payload := req.JSON(t)
		assert.Equal(t, channel, payload["channel"])
		assert.Equal(t, "watchtower", payload["username"])
		assert.Equal(t, ":whale:", payload["icon_emoji"])
		assert.Equal(t, "*"+testMessage.Title+"*\n"+testMessage.Body, payload["text"])
	}
}

func TestSlackNativeWebhook(t *testing.T) {
	server := newTestServer(t)
	send(t, "https://hooks.slack.com/services/T000/B000/XXXX", server)

	req := server.only(t)
	assert.Equal(t, "https://hooks.slack.com/services/T000/B000/XXXX", req.URL)
	assert.NotContains(t, req.JSON(t), "channel")
}

func TestSlackLegacyImageParameter(t *testing.T) {
	service, err := Parse("slack://bot@T/B/C?image=https://example.com/icon.png", Options{})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/icon.png", service.(*slack).iconURL)
}

func TestSlackBot(t *testing.T) {
	server := newTestServer(t)
	send(t, "slack://xoxb-1234-abcd/general", server)

	req := server.only(t)
	assert.Equal(t, "https://slack.com/api/chat.postMessage", req.URL)
	assert.Equal(t, "Bearer xoxb-1234-abcd", req.Header.Get("Authorization"))
	assert.Equal(t, "#general", req.JSON(t)["channel"])

	server.responder = func(w http.ResponseWriter, _ int) {
		_, _ = w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
	}
	service, err := Parse("slack://xoxb-1234-abcd/general", server.options())
	require.NoError(t, err)
	assert.ErrorContains(t, service.Send(context.Background(), testMessage), "channel_not_found")
}

func TestDiscord(t *testing.T) {
	server := newTestServer(t)
	send(t, "discord://Watchtower@123456/abc-token?avatar_url=https://example.com/a.png&thread=42", server)

	req := server.only(t)
	assert.Equal(t, "https://discord.com/api/webhooks/123456/abc-token?thread_id=42&wait=true", req.URL)
	payload := req.JSON(t)
	assert.Equal(t, "Watchtower", payload["username"])
	assert.Equal(t, "https://example.com/a.png", payload["avatar_url"])
	embed := payload["embeds"].([]any)[0].(map[string]any)
	assert.Equal(t, testMessage.Title, embed["title"])
	assert.Equal(t, testMessage.Body, embed["description"])
}

func TestDiscordNativeAndLegacyParameters(t *testing.T) {
	server := newTestServer(t)
	send(t, "https://discordapp.com/api/webhooks/123456/abc-token?username=bot&avatar=https://example.com/a.png", server)

	req := server.only(t)
	assert.Equal(t, "https://discord.com/api/webhooks/123456/abc-token?wait=true", req.URL)
	assert.Equal(t, "bot", req.JSON(t)["username"])
	assert.Equal(t, "https://example.com/a.png", req.JSON(t)["avatar_url"])
}

func TestTelegram(t *testing.T) {
	server := newTestServer(t)
	send(t, "tgram://123456789:AAbbCC/-100200/@channel?silent=yes&topic=7", server)

	requests := server.all()
	require.Len(t, requests, 2)
	for i, chatID := range []string{"-100200", "@channel"} {
		req := requests[i]
		assert.Equal(t, "https://api.telegram.org/bot123456789:AAbbCC/sendMessage", req.URL)
		payload := req.JSON(t)
		assert.Equal(t, chatID, payload["chat_id"])
		assert.Equal(t, testMessage.Title+"\n"+testMessage.Body, payload["text"])
		assert.Equal(t, true, payload["disable_notification"])
		assert.EqualValues(t, 7, payload["message_thread_id"])
	}
}

func TestTelegramAPIError(t *testing.T) {
	server := newTestServer(t)
	server.responder = func(w http.ResponseWriter, _ int) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: chat not found"}`))
	}
	service, err := Parse("tgram://bot123456789:AAbbCC/42", server.options())
	require.NoError(t, err)
	err = service.Send(context.Background(), testMessage)
	assert.ErrorContains(t, err, "chat not found")
	assert.NotContains(t, err.Error(), "AAbbCC", "errors should not expose the bot token")
}

func TestPushover(t *testing.T) {
	server := newTestServer(t)
	send(t, "pover://user-key@app-token/phone/tablet?priority=emergency&sound=siren", server)

	req := server.only(t)
	assert.Equal(t, "https://api.pushover.net/1/messages.json", req.URL)
	values, err := url.ParseQuery(string(req.Body))
	require.NoError(t, err)
	assert.Equal(t, "app-token", values.Get("token"))
	assert.Equal(t, "user-key", values.Get("user"))
	assert.Equal(t, testMessage.Body, values.Get("message"))
	assert.Equal(t, testMessage.Title, values.Get("title"))
	assert.Equal(t, "phone,tablet", values.Get("device"))
	assert.Equal(t, "2", values.Get("priority"))
	assert.Equal(t, "siren", values.Get("sound"))
	assert.Equal(t, "60", values.Get("retry"))
}

func TestTeams(t *testing.T) {
	for rawURL, expected := range map[string]string{
		"msteams://aaa@bbb/ccc/ddd/":            "https://outlook.office.com/webhook/aaa@bbb/IncomingWebhook/ccc/ddd",
		"msteams://contoso/aaa@bbb/ccc/ddd/eee": "https://contoso.webhook.office.com/webhookb2/aaa@bbb/IncomingWebhook/ccc/ddd/eee",
		"https://contoso.webhook.office.com/webhookb2/aaa@bbb/IncomingWebhook/ccc/ddd/eee": "https://contoso.webhook.office.com/webhookb2/aaa@bbb/IncomingWebhook/ccc/ddd/eee",
	} {
		server := newTestServer(t)
		send(t, rawURL, server)

		req := server.only(t)
		assert.Equal(t, expected, req.URL, rawURL)
		payload := req.JSON(t)
		assert.Equal(t, "MessageCard", payload["@type"])
		assert.Equal(t, testMessage.Title, payload["title"])
		assert.Equal(t, "Updated app (app:latest)<br>Found 2 updates", payload["text"])
	}
}

func TestWorkflows(t *testing.T) {
	server := newTestServer(t)
	send(t, "workflows://prod-01.westus.logic.azure.com:443/wf-id/sig-value", server)

	req := server.only(t)
	endpoint, err := url.Parse(req.URL)
	require.NoError(t, err)
	assert.Equal(t, "prod-01.westus.logic.azure.com:443", endpoint.Host)
	assert.Equal(t, "/workflows/wf-id/triggers/manual/paths/invoke", endpoint.Path)
	assert.Equal(t, "sig-value", endpoint.Query().Get("sig"))
	assert.Equal(t, "/triggers/manual/run", endpoint.Query().Get("sp"))

	payload := req.JSON(t)
	card := payload["attachments"].([]any)[0].(map[string]any)
	assert.Equal(t, "application/vnd.microsoft.card.adaptive", card["contentType"])
	body := card["content"].(map[string]any)["body"].([]any)
	assert.Len(t, body, 2)
	assert.Equal(t, testMessage.Body, body[1].(map[string]any)["text"])
}

func TestWorkflowsNativeURL(t *testing.T) {
	rawURL := "https://default123.environment.api.powerplatform.com:443/powerautomate/automations/direct/workflows/abc/triggers/manual/paths/invoke?api-version=1&sig=xyz"
	server := newTestServer(t)
	send(t, rawURL, server)
	assert.Equal(t, rawURL, server.only(t).URL)
}

func TestJSON(t *testing.T) {
	server := newTestServer(t)
	send(t, "jsons://user:pass@hooks.example.com:8443/notify/watchtower?method=put&+X-Api-Key=secret", server)

	req := server.only(t)
	assert.Equal(t, http.MethodPut, req.Method)
	assert.Equal(t, "https://hooks.example.com:8443/notify/watchtower", req.URL)
	assert.Equal(t, "secret", req.Header.Get("X-Api-Key"))
	assert.Equal(t, basicAuth("user", "pass"), req.Header.Get("Authorization"))
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	payload := req.JSON(t)
	assert.Equal(t, "1.0", payload["version"])
	assert.Equal(t, "info", payload["type"])
	assert.Equal(t, testMessage.Title, payload["title"])
	assert.Equal(t, testMessage.Body, payload["message"])
}

func TestRateLimitRetry(t *testing.T) {
	server := newTestServer(t)
	server.responder = func(w http.ResponseWriter, attempt int) {
		if attempt == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
	send(t, "discord://1/token", server)
	assert.Len(t, server.all(), 2)
}

func TestHTTPErrors(t *testing.T) {
	server := newTestServer(t)
	server.responder = func(w http.ResponseWriter, _ int) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized",   "errorDescription": "invalid token"}`))
	}
	service, err := Parse("gotify://gotify/token", server.options())
	require.NoError(t, err)
	err = service.Send(context.Background(), testMessage)
	assert.EqualError(t, err, `gotify responded with 401 Unauthorized: {"error":"unauthorized", "errorDescription": "invalid token"}`)
}

func TestConnectionErrorsHideTokens(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	address := strings.TrimPrefix(server.URL, "http://")
	server.Close()

	service, err := Parse("json://"+address+"/hook/super-secret-token", Options{})
	require.NoError(t, err)
	err = service.Send(context.Background(), testMessage)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "super-secret-token")
	assert.Contains(t, err.Error(), address)
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "short", truncate("short", 10))
	assert.Equal(t, "åäö…", truncate("åäöåäö", 4))
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()

	textPath := filepath.Join(dir, "apprise.txt")
	require.NoError(t, os.WriteFile(textPath, []byte(`
# comment
; also a comment
gotifys://gotify.example.com/token
admins, ops = mailtos://user:pass@example.com?to=a@example.com
`), 0o600))
	urls, err := LoadConfig(textPath)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"gotifys://gotify.example.com/token",
		"mailtos://user:pass@example.com?to=a@example.com",
	}, urls)

	yamlPath := filepath.Join(dir, "apprise.yml")
	require.NoError(t, os.WriteFile(yamlPath, []byte(`
version: 1
urls:
  - ntfys://ntfy.example.com/topic
  - "tgram://123:abc/456":
      - tag: ops
`), 0o600))
	urls, err = LoadConfig(yamlPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"ntfys://ntfy.example.com/topic", "tgram://123:abc/456"}, urls)

	_, err = LoadConfig(filepath.Join(dir, "missing.yml"))
	assert.Error(t, err)
}
