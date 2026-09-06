package notifications

import (
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/cobra"

	t "github.com/containrrr/watchtower/pkg/types"
	log "github.com/sirupsen/logrus"
)

const (
	emailType = "email"
)

type emailTypeNotifier struct {
	From, To               string
	Server, User, Password string
	Port                   int
	tlsSkipVerify          bool
	entries                []*log.Entry
	delay                  time.Duration
}

func newEmailNotifier(c *cobra.Command) t.ConvertibleNotifier {
	flags := c.Flags()

	from, _ := flags.GetString("notification-email-from")
	to, _ := flags.GetString("notification-email-to")
	server, _ := flags.GetString("notification-email-server")
	user, _ := flags.GetString("notification-email-server-user")
	password, _ := flags.GetString("notification-email-server-password")
	port, _ := flags.GetInt("notification-email-server-port")
	tlsSkipVerify, _ := flags.GetBool("notification-email-server-tls-skip-verify")
	delay, _ := flags.GetInt("notification-email-delay")

	n := &emailTypeNotifier{
		entries:       []*log.Entry{},
		From:          from,
		To:            to,
		Server:        server,
		User:          user,
		Password:      password,
		Port:          port,
		tlsSkipVerify: tlsSkipVerify,
		delay:         time.Duration(delay) * time.Second,
	}

	return n
}

func (e *emailTypeNotifier) GetURL(c *cobra.Command) (string, error) {
	scheme := "mailtos"
	if e.Port == 25 || e.tlsSkipVerify {
		scheme = "mailto"
	}

	u := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", e.Server, e.Port),
	}

	if e.User != "" || e.Password != "" {
		u.User = url.UserPassword(e.User, e.Password)
	}

	q := url.Values{}
	q.Set("from", fmt.Sprintf("Watchtower <%s>", e.From))
	q.Set("to", e.To)
	if e.Server != "" {
		q.Set("smtp", e.Server)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func (e *emailTypeNotifier) GetDelay() time.Duration {
	return e.delay
}
