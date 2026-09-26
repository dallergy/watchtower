package apprise

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// smtpServer is a minimal SMTP server that accepts a single session without TLS
type smtpServer struct {
	listener net.Listener
	mu       sync.Mutex
	auth     string
	from     string
	rcpt     []string
	data     string
	done     chan struct{}
}

func newSMTPServer(t *testing.T, extensions ...string) *smtpServer {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := &smtpServer{listener: listener, done: make(chan struct{})}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		defer close(s.done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		s.serve(conn, extensions)
	}()
	return s
}

func (s *smtpServer) port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

func (s *smtpServer) serve(conn net.Conn, extensions []string) {
	reader := bufio.NewReader(conn)
	reply := func(line string) { _, _ = fmt.Fprintf(conn, "%s\r\n", line) }
	reply("220 test ESMTP")

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.TrimSpace(line)
		upper := strings.ToUpper(command)

		s.mu.Lock()
		switch {
		case strings.HasPrefix(upper, "EHLO"):
			lines := append([]string{"test"}, extensions...)
			for i, ext := range lines {
				separator := "-"
				if i == len(lines)-1 {
					separator = " "
				}
				reply("250" + separator + ext)
			}
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			decoded, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(command[len("AUTH PLAIN"):]))
			s.auth = string(decoded)
			reply("235 accepted")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			s.from = strings.Trim(command[len("MAIL FROM:"):], "<> ")
			reply("250 ok")
		case strings.HasPrefix(upper, "RCPT TO:"):
			s.rcpt = append(s.rcpt, strings.Trim(command[len("RCPT TO:"):], "<> "))
			reply("250 ok")
		case upper == "DATA":
			reply("354 go ahead")
			var data strings.Builder
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil || dataLine == ".\r\n" {
					break
				}
				data.WriteString(dataLine)
			}
			s.data = data.String()
			reply("250 queued")
		case upper == "QUIT":
			reply("221 bye")
			s.mu.Unlock()
			return
		default:
			reply("502 unsupported")
		}
		s.mu.Unlock()
	}
}

func TestEmail(t *testing.T) {
	server := newSMTPServer(t, "AUTH PLAIN LOGIN")
	rawURL := fmt.Sprintf("mailto://bot:secret@127.0.0.1:%d?from=watchtower@example.com&name=Watchtower+Bot&to=ops@example.com,dev@example.com&bcc=audit@example.com", server.port())

	service, err := Parse(rawURL, Options{})
	require.NoError(t, err)
	require.NoError(t, service.Send(context.Background(), testMessage))
	<-server.done

	assert.Equal(t, "\x00bot\x00secret", server.auth)
	assert.Equal(t, "watchtower@example.com", server.from)
	assert.Equal(t, []string{"ops@example.com", "dev@example.com", "audit@example.com"}, server.rcpt)

	msg, err := mail.ReadMessage(strings.NewReader(server.data))
	require.NoError(t, err)
	assert.Equal(t, `"Watchtower Bot" <watchtower@example.com>`, msg.Header.Get("From"))
	assert.Equal(t, "ops@example.com, dev@example.com", msg.Header.Get("To"))
	assert.Empty(t, msg.Header.Get("Bcc"))
	subject, err := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	require.NoError(t, err)
	assert.Equal(t, testMessage.Title, subject)
	assert.Contains(t, server.data, "Updated app (app:latest)\r\nFound 2 updates")
}

func TestEmailDefaults(t *testing.T) {
	service, err := Parse("mailtos://user@example.com:secret@smtp.example.com", Options{})
	require.NoError(t, err)
	e := service.(*email)
	assert.Equal(t, "smtp.example.com", e.server)
	assert.Equal(t, 587, e.port)
	assert.Equal(t, emailModeStartTLS, e.mode)
	assert.Equal(t, "user@example.com", e.from.Address)
	assert.Equal(t, []string{"user@example.com"}, e.to)

	service, err = Parse("mailtos://user:secret@example.com:465/to@example.com?smtp=mail.example.com", Options{})
	require.NoError(t, err)
	e = service.(*email)
	assert.Equal(t, "mail.example.com", e.server)
	assert.Equal(t, emailModeSSL, e.mode)
	assert.Equal(t, "user@example.com", e.from.Address)
	assert.Equal(t, []string{"to@example.com"}, e.to)

	service, err = Parse("mailto://mail.local?to=admin@example.com", Options{})
	require.NoError(t, err)
	e = service.(*email)
	assert.Equal(t, 25, e.port)
	assert.Equal(t, emailModeAuto, e.mode)
	assert.Equal(t, "watchtower@mail.local", e.from.Address)
}

func TestEmailRequiresTLS(t *testing.T) {
	server := newSMTPServer(t, "AUTH PLAIN")
	service, err := Parse(fmt.Sprintf("mailtos://bot:secret@127.0.0.1:%d?to=ops@example.com", server.port()), Options{})
	require.NoError(t, err)
	assert.ErrorContains(t, service.Send(context.Background(), testMessage), "does not support STARTTLS")
}

func TestEmailRefusesUnencryptedCredentials(t *testing.T) {
	server := newSMTPServer(t, "AUTH PLAIN")
	e := &email{
		server:   "127.0.0.1",
		port:     server.port(),
		mode:     emailModeAuto,
		user:     "bot",
		password: "secret",
	}
	client, err := e.connect(context.Background())
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	// Pretend the server is remote to exercise the check
	e.server = "smtp.example.com"
	assert.ErrorContains(t, e.authenticate(client), "unencrypted")
}
