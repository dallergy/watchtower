package apprise

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// serviceURL is a leniently parsed Apprise URL.
//
// Apprise URLs are not always valid RFC 3986 URLs: Telegram bot tokens contain colons, Slack
// channels are prefixed with '#' and Teams tokens contain '@'. They are therefore split by hand
// instead of using net/url, and '#' is never treated as the start of a fragment.
type serviceURL struct {
	scheme string
	// user and password hold the unescaped user info, if any
	user        string
	password    string
	hasUserInfo bool
	// host is the unescaped authority without user info, which may be a token rather than a host name
	host string
	// path holds the unescaped, non-empty path segments
	path []string
	// params holds the query parameters with lower-cased keys
	params map[string]string
	// headers holds the query parameters prefixed with '+', which set HTTP headers
	headers map[string]string
}

func parseServiceURL(rawURL string) (*serviceURL, error) {
	scheme, rest, found := strings.Cut(strings.TrimSpace(rawURL), "://")
	if !found || scheme == "" {
		return nil, fmt.Errorf("missing scheme")
	}

	u := &serviceURL{
		scheme:  strings.ToLower(scheme),
		params:  map[string]string{},
		headers: map[string]string{},
	}

	rest, rawQuery, _ := strings.Cut(rest, "?")
	if err := u.parseQuery(rawQuery); err != nil {
		return nil, err
	}

	authority, rawPath, _ := strings.Cut(rest, "/")
	if at := strings.LastIndex(authority, "@"); at >= 0 {
		u.hasUserInfo = true
		rawUser, rawPassword, hasPassword := strings.Cut(authority[:at], ":")
		var err error
		if u.user, err = url.PathUnescape(rawUser); err != nil {
			return nil, fmt.Errorf("invalid user: %w", err)
		}
		if hasPassword {
			if u.password, err = url.PathUnescape(rawPassword); err != nil {
				return nil, fmt.Errorf("invalid password: %w", err)
			}
		}
		authority = authority[at+1:]
	}

	host, err := url.PathUnescape(authority)
	if err != nil {
		return nil, fmt.Errorf("invalid host: %w", err)
	}
	u.host = host

	for _, rawSegment := range strings.Split(rawPath, "/") {
		if rawSegment == "" {
			continue
		}
		segment, err := url.PathUnescape(rawSegment)
		if err != nil {
			return nil, fmt.Errorf("invalid path: %w", err)
		}
		u.path = append(u.path, segment)
	}

	return u, nil
}

func (u *serviceURL) parseQuery(rawQuery string) error {
	for _, pair := range strings.Split(rawQuery, "&") {
		if pair == "" {
			continue
		}
		rawKey, rawValue, _ := strings.Cut(pair, "=")
		value, err := url.QueryUnescape(rawValue)
		if err != nil {
			return fmt.Errorf("invalid query parameter: %w", err)
		}
		if strings.HasPrefix(rawKey, "+") {
			header, err := url.PathUnescape(rawKey[1:])
			if err != nil {
				return fmt.Errorf("invalid header parameter: %w", err)
			}
			u.headers[header] = value
			continue
		}
		key, err := url.QueryUnescape(rawKey)
		if err != nil {
			return fmt.Errorf("invalid query parameter: %w", err)
		}
		u.params[strings.ToLower(key)] = value
	}
	return nil
}

// hostPort splits the host into a host name and an optional port (0 when absent)
func (u *serviceURL) hostPort() (string, int, error) {
	host := u.host
	if host == "" {
		return "", 0, fmt.Errorf("missing host")
	}

	hostname, rawPort, err := net.SplitHostPort(host)
	if err != nil {
		// no port present, or an IPv6 address without one
		return strings.Trim(host, "[]"), 0, nil
	}

	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("invalid port %q", rawPort)
	}
	return hostname, port, nil
}

// baseURL returns the web address of the host, using TLS when secure is set
func (u *serviceURL) baseURL(secure bool) (*url.URL, error) {
	hostname, port, err := u.hostPort()
	if err != nil {
		return nil, err
	}

	base := &url.URL{Scheme: "http", Host: hostname}
	if secure {
		base.Scheme = "https"
	}
	if strings.Contains(hostname, ":") {
		base.Host = "[" + hostname + "]"
	}
	if port > 0 {
		base.Host = net.JoinHostPort(hostname, strconv.Itoa(port))
	}
	return base, nil
}

// param returns the first non-empty query parameter among the given keys
func (u *serviceURL) param(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(u.params[key]); value != "" {
			return value
		}
	}
	return ""
}

// boolParam parses a yes/no style query parameter, returning fallback when it is absent or invalid
func (u *serviceURL) boolParam(key string, fallback bool) bool {
	switch strings.ToLower(u.param(key)) {
	case "yes", "y", "true", "on", "1", "enable", "enabled":
		return true
	case "no", "n", "false", "off", "0", "disable", "disabled":
		return false
	default:
		return fallback
	}
}

// listParam splits a comma or space separated query parameter into its values
func (u *serviceURL) listParam(keys ...string) []string {
	return splitList(u.param(keys...))
}

func splitList(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})
}
