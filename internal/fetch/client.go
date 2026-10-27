// Package fetch loads user supplied URLs on the server without turning it into a proxy into its own network
// (SSRF): every redirect hop is checked against the address rules and responses are size limited.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

const (
	timeout      = 10 * time.Second
	maxRedirects = 5
	userAgent    = "ragbag/1.0"
)

var (
	ErrUnresolvable = errors.New("host cannot be resolved")
	ErrPrivate      = errors.New("private network address")
	ErrForbidden    = errors.New("loopback or link-local address")
	ErrTooLarge     = errors.New("response too large")
)

type Client struct {
	allowPrivate bool
	http         *http.Client
}

// NewClient refuses loopback and link-local (cloud metadata) addresses always, private networks unless
// allowPrivate is set (homelab dashboards).
func NewClient(allowPrivate bool) *Client {
	c := &Client{allowPrivate: allowPrivate}
	dialer := &net.Dialer{
		Timeout: timeout,
		// Checks the address actually connected to, so DNS rebinding between check and connect does not help
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			return c.checkIP(net.ParseIP(host))
		},
	}
	c.http = &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:         dialer.DialContext,
			TLSHandshakeTimeout: timeout,
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > maxRedirects {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errors.New("redirect to a non-http URL")
			}
			return c.checkHost(req.Context(), req.URL.Hostname())
		},
	}
	return c
}

type Response struct {
	URL         *url.URL // after redirects
	ContentType string
	Body        []byte
}

// Get loads the URL and returns the body of a 2xx response up to maxBytes.
func (c *Client) Get(ctx context.Context, rawURL string, maxBytes int64) (*Response, error) {
	target, err := url.Parse(rawURL)
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Hostname() == "" {
		return nil, fmt.Errorf("not an http(s) URL: %q", rawURL)
	}
	if err := c.checkHost(ctx, target.Hostname()); err != nil {
		c.logRefusal(target, err)
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	res, err := c.http.Do(req)
	if err != nil {
		var refusal error
		for _, known := range []error{ErrUnresolvable, ErrPrivate, ErrForbidden} {
			if errors.Is(err, known) {
				refusal = known
			}
		}
		if refusal != nil {
			c.logRefusal(target, refusal)
			return nil, refusal
		}
		slog.Warn("fetch failed", "url", rawURL, "error", err)
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, ErrTooLarge
	}
	return &Response{URL: res.Request.URL, ContentType: res.Header.Get("Content-Type"), Body: body}, nil
}

func (c *Client) logRefusal(target *url.URL, err error) {
	switch {
	case errors.Is(err, ErrUnresolvable):
		// Most often DNS inside the container is broken (e.g. a host firewall blocking the network's DNS)
		slog.Warn("cannot resolve host; check DNS resolution of the server/container", "url", target.String())
	case errors.Is(err, ErrPrivate):
		slog.Warn("refusing private address; set ALLOW_PRIVATE_HOSTS=true to allow your own network", "url", target.String())
	default:
		slog.Warn("refusing loopback/link-local address", "url", target.String())
	}
}

func (c *Client) checkHost(ctx context.Context, host string) error {
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(addresses) == 0 {
		return ErrUnresolvable
	}
	for _, address := range addresses {
		if err := c.checkIP(address.IP); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) checkIP(ip net.IP) error {
	if ip == nil {
		return ErrUnresolvable
	}
	// Never reachable: the server itself and link-local, which includes 169.254.169.254 (cloud metadata)
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return ErrForbidden
	}
	if !c.allowPrivate && ip.IsPrivate() {
		return ErrPrivate
	}
	return nil
}

// NormalizeURL turns user input into an absolute http(s) URL, adding https:// if the scheme is missing.
// Anything else (e.g. javascript:) would be rendered as a clickable link for every member of a space.
func NormalizeURL(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New("URL must not be empty")
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("Invalid URL: %s", value)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if (scheme != "http" && scheme != "https") || parsed.Hostname() == "" {
		return "", errors.New("Only http and https URLs are supported")
	}
	return trimmed, nil
}
