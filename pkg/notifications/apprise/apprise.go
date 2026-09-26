// Package apprise sends notifications to a focused subset of the services supported by
// Apprise (https://github.com/caronc/apprise), using the same URL syntax, natively in Go.
//
// No Apprise installation, Python runtime or sidecar container is needed. Services that are
// not implemented here can still be reached through an external Apprise API server.
package apprise

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// Message is a notification to deliver
type Message struct {
	Title string
	Body  string
}

// Service is a notification service created from an Apprise URL
type Service interface {
	// Scheme returns the URL scheme that identifies the service, e.g. "gotify"
	Scheme() string
	// Send delivers the message to the service
	Send(ctx context.Context, msg Message) error
}

// Options modify how services are created
type Options struct {
	// Transport overrides the HTTP transport used by web based services
	Transport http.RoundTripper
}

type factory func(u *serviceURL, opts Options) (Service, error)

var factories = map[string]factory{
	"discord":   newDiscord,
	"gotify":    newGotify,
	"gotifys":   newGotify,
	"json":      newJSON,
	"jsons":     newJSON,
	"mailto":    newEmail,
	"mailtos":   newEmail,
	"msteams":   newTeams,
	"ntfy":      newNtfy,
	"ntfys":     newNtfy,
	"pover":     newPushover,
	"slack":     newSlack,
	"tgram":     newTelegram,
	"workflow":  newWorkflows,
	"workflows": newWorkflows,
}

// UnsupportedError is returned when a URL uses a scheme that is not implemented natively
type UnsupportedError struct {
	Scheme string
}

func (e UnsupportedError) Error() string {
	return fmt.Sprintf("the %q notification service is not built in (supported: %s)", e.Scheme, strings.Join(Schemes(), ", "))
}

// Schemes returns the sorted list of URL schemes that are supported natively
func Schemes() []string {
	schemes := make([]string, 0, len(factories))
	for scheme := range factories {
		schemes = append(schemes, scheme)
	}
	slices.Sort(schemes)
	return schemes
}

// Scheme returns the lower-cased scheme of a notification URL, or an empty string if it has none
func Scheme(rawURL string) string {
	scheme, _, found := strings.Cut(strings.TrimSpace(rawURL), "://")
	if !found {
		return ""
	}
	return strings.ToLower(scheme)
}

// IsSupported returns whether the URL can be handled natively by Parse
func IsSupported(rawURL string) bool {
	scheme := Scheme(rawURL)
	if scheme == "http" || scheme == "https" {
		_, err := parseNative(rawURL, Options{})
		return err == nil
	}
	_, found := factories[scheme]
	return found
}

// Parse creates the Service described by an Apprise URL
func Parse(rawURL string, opts Options) (Service, error) {
	rawURL = strings.TrimSpace(rawURL)
	scheme := Scheme(rawURL)
	if scheme == "" {
		return nil, fmt.Errorf("invalid notification URL: missing scheme")
	}

	if scheme == "http" || scheme == "https" {
		return parseNative(rawURL, opts)
	}

	create, found := factories[scheme]
	if !found {
		return nil, UnsupportedError{Scheme: scheme}
	}

	u, err := parseServiceURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid %s URL: %w", scheme, err)
	}

	service, err := create(u, opts)
	if err != nil {
		return nil, fmt.Errorf("invalid %s URL: %w", scheme, err)
	}
	return service, nil
}

// parseNative creates services from the webhook URLs issued by the services themselves
func parseNative(rawURL string, opts Options) (Service, error) {
	u, err := parseServiceURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid notification URL: %w", err)
	}
	hostname, _, _ := u.hostPort()
	hostname = strings.ToLower(hostname)

	switch {
	case hostname == "hooks.slack.com" && len(u.path) >= 4 && u.path[0] == "services":
		return newSlackWebhook(u, u.path[1:4], nil, opts), nil
	case (hostname == "discord.com" || hostname == "discordapp.com") &&
		len(u.path) >= 4 && u.path[0] == "api" && u.path[1] == "webhooks":
		return newDiscordWebhook(u, u.path[2], u.path[3], opts), nil
	case strings.HasSuffix(hostname, "office.com") && len(u.path) > 0 && strings.HasPrefix(u.path[0], "webhook"):
		return newTeamsWebhook(u, rawURL, opts), nil
	case slices.Contains(u.path, "workflows") && slices.Contains(u.path, "triggers"):
		return newWorkflowsWebhook(u, rawURL, opts), nil
	}

	return nil, UnsupportedError{Scheme: u.scheme}
}
