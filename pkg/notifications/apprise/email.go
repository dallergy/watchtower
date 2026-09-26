package apprise

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// email sends messages over SMTP.
//
//	mailto://[user:password@]hostname[:port]?to=address    (STARTTLS when offered, port 25)
//	mailtos://[user:password@]hostname[:port]?to=address   (TLS required, port 587, or 465 for implicit TLS)
//
// Parameters: to, cc, bcc, from, name (sender name), smtp (server, when it differs from the
// hostname), mode (starttls, ssl or insecure), verify (yes/no)
type email struct {
	server    string
	port      int
	mode      string
	user      string
	password  string
	from      mail.Address
	to        []string
	cc        []string
	bcc       []string
	tlsConfig *tls.Config
}

const (
	emailModeAuto     = "auto"
	emailModeStartTLS = "starttls"
	emailModeSSL      = "ssl"
	emailModeInsecure = "insecure"
	emailTimeout      = time.Minute
)

func newEmail(u *serviceURL, _ Options) (Service, error) {
	hostname, port, err := u.hostPort()
	if err != nil {
		return nil, err
	}

	e := &email{
		server:   hostname,
		port:     port,
		mode:     emailModeAuto,
		user:     u.user,
		password: u.password,
	}
	if server := u.param("smtp"); server != "" {
		e.server = server
	}
	if user := u.param("user"); user != "" {
		e.user = user
	}
	if password := u.param("pass", "password"); password != "" {
		e.password = password
	}

	if u.scheme == "mailtos" {
		e.mode = emailModeStartTLS
		if e.port == 465 {
			e.mode = emailModeSSL
		}
	}
	if mode := strings.ToLower(u.param("mode")); mode != "" {
		switch mode {
		case emailModeStartTLS, emailModeSSL, emailModeInsecure:
			e.mode = mode
		default:
			return nil, fmt.Errorf("invalid mode %q (expected starttls, ssl or insecure)", mode)
		}
	}
	if e.port == 0 {
		e.port = defaultEmailPort(e.mode)
	}

	e.tlsConfig = &tls.Config{ServerName: e.server}
	if !u.boolParam("verify", true) {
		e.tlsConfig.InsecureSkipVerify = true //nolint:gosec // explicitly requested using verify=no
	}

	if err := e.parseAddresses(u, hostname); err != nil {
		return nil, err
	}
	return e, nil
}

func defaultEmailPort(mode string) int {
	switch mode {
	case emailModeSSL:
		return 465
	case emailModeStartTLS:
		return 587
	default:
		return 25
	}
}

func (e *email) parseAddresses(u *serviceURL, hostname string) error {
	rawFrom := u.param("from")
	if rawFrom == "" {
		rawFrom = e.user
		if !strings.Contains(rawFrom, "@") {
			localPart := rawFrom
			if localPart == "" {
				localPart = "watchtower"
			}
			rawFrom = localPart + "@" + hostname
		}
	}
	from, err := mail.ParseAddress(rawFrom)
	if err != nil {
		return fmt.Errorf("invalid from address %q: %w", rawFrom, err)
	}
	if name := u.param("name"); name != "" {
		from.Name = name
	}
	if from.Name == "" {
		from.Name = "Watchtower"
	}
	e.from = *from

	// Recipients can also be given as path segments, e.g. mailto://user:pass@example.com/to@example.com
	recipients := append(u.listParam("to"), u.path...)
	if len(recipients) == 0 {
		recipients = []string{from.Address}
	}
	if e.to, err = parseAddressList(recipients); err != nil {
		return err
	}
	if e.cc, err = parseAddressList(u.listParam("cc")); err != nil {
		return err
	}
	e.bcc, err = parseAddressList(u.listParam("bcc"))
	return err
}

func parseAddressList(values []string) ([]string, error) {
	addresses := make([]string, 0, len(values))
	for _, value := range values {
		address, err := mail.ParseAddress(value)
		if err != nil {
			return nil, fmt.Errorf("invalid address %q: %w", value, err)
		}
		addresses = append(addresses, address.Address)
	}
	return addresses, nil
}

func (e *email) Scheme() string { return "mailto" }

func (e *email) Send(ctx context.Context, msg Message) error {
	ctx, cancel := context.WithTimeout(ctx, emailTimeout)
	defer cancel()

	client, err := e.connect(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	if err := e.deliver(client, msg); err != nil {
		return err
	}
	return client.Quit()
}

func (e *email) connect(ctx context.Context) (*smtp.Client, error) {
	address := net.JoinHostPort(e.server, strconv.Itoa(e.port))
	dialer := &net.Dialer{}

	var conn net.Conn
	var err error
	if e.mode == emailModeSSL {
		tlsDialer := &tls.Dialer{NetDialer: dialer, Config: e.tlsConfig}
		conn, err = tlsDialer.DialContext(ctx, "tcp", address)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, e.server)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to start SMTP session with %s: %w", address, err)
	}
	return client, nil
}

func (e *email) deliver(client *smtp.Client, msg Message) error {
	if e.mode == emailModeAuto || e.mode == emailModeStartTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(e.tlsConfig); err != nil {
				return fmt.Errorf("failed to start TLS: %w", err)
			}
		} else if e.mode == emailModeStartTLS {
			return errors.New("the SMTP server does not support STARTTLS (use mailto:// or mode=insecure to allow unencrypted mail)")
		}
	}

	if e.user != "" {
		if err := e.authenticate(client); err != nil {
			return err
		}
	}

	if err := client.Mail(e.from.Address); err != nil {
		return fmt.Errorf("sender rejected: %w", err)
	}
	for _, recipient := range slices.Concat(e.to, e.cc, e.bcc) {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("recipient %s rejected: %w", recipient, err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(e.buildMessage(msg)); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

func (e *email) authenticate(client *smtp.Client) error {
	ok, mechanisms := client.Extension("AUTH")
	if !ok {
		return errors.New("credentials were given, but the SMTP server does not support authentication")
	}
	if _, isTLS := client.TLSConnectionState(); !isTLS && e.mode != emailModeInsecure && !isLoopback(e.server) {
		return errors.New("refusing to send credentials over an unencrypted connection (use mode=insecure to allow it)")
	}

	var auth smtp.Auth
	if strings.Contains(strings.ToUpper(mechanisms), "PLAIN") {
		auth = &plainAuth{user: e.user, password: e.password}
	} else {
		auth = &loginAuth{user: e.user, password: e.password}
	}
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	return nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (e *email) buildMessage(msg Message) []byte {
	var buffer bytes.Buffer
	header := func(key, value string) {
		buffer.WriteString(key + ": " + value + "\r\n")
	}

	header("From", e.from.String())
	header("To", strings.Join(e.to, ", "))
	if len(e.cc) > 0 {
		header("Cc", strings.Join(e.cc, ", "))
	}
	header("Subject", mime.QEncoding.Encode("utf-8", msg.Title))
	header("Date", time.Now().Format(time.RFC1123Z))
	header("Message-ID", e.messageID())
	header("MIME-Version", "1.0")
	header("Content-Type", "text/plain; charset=utf-8")
	header("Content-Transfer-Encoding", "quoted-printable")
	buffer.WriteString("\r\n")

	body := quotedprintable.NewWriter(&buffer)
	_, _ = body.Write([]byte(msg.Body))
	_ = body.Close()
	buffer.WriteString("\r\n")
	return buffer.Bytes()
}

func (e *email) messageID() string {
	random := make([]byte, 12)
	_, _ = rand.Read(random)
	domain := e.server
	if _, fromDomain, found := strings.Cut(e.from.Address, "@"); found {
		domain = fromDomain
	}
	return "<" + hex.EncodeToString(random) + "@" + domain + ">"
}

// plainAuth implements the PLAIN mechanism. Unlike smtp.PlainAuth it leaves the decision about
// encryption to authenticate, so that mode=insecure can be honoured.
type plainAuth struct {
	user, password string
}

func (a *plainAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "PLAIN", []byte("\x00" + a.user + "\x00" + a.password), nil
}

func (a *plainAuth) Next(_ []byte, more bool) ([]byte, error) {
	if more {
		return nil, errors.New("unexpected server challenge")
	}
	return nil, nil
}

// loginAuth implements the LOGIN mechanism, which some servers offer instead of PLAIN
type loginAuth struct {
	user, password string
}

func (a *loginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(challenge []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(string(challenge))) {
	case "username:":
		return []byte(a.user), nil
	case "password:":
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("unexpected server challenge %q", challenge)
	}
}
