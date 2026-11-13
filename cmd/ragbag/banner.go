package main

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/Serbroda/ragbag/internal/config"
)

const logo = `
                   _
 _ __ __ _  __ _  | |__   __ _  __ _
| '__/ _` + "`" + ` |/ _` + "`" + ` | | '_ \ / _` + "`" + ` |/ _` + "`" + ` |
| | | (_| | (_| | | |_) | (_| | (_| |
|_|  \__,_|\__, | |_.__/ \__,_|\__, |
           |___/               |___/
`

// banner is the start message written once the server listens: from here on requests are answered.
func banner(cfg config.Config, version string, listener net.Addr) string {
	var b strings.Builder
	b.WriteString(logo)
	b.WriteString("\n")
	row := func(label, value string) { fmt.Fprintf(&b, "  %-14s %s\n", label, value) }
	row("Version", version)
	row("Listening on", listenAddress(listener))
	row("Address", orNot(cfg.PublicURL, "not set (PUBLIC_URL)"))
	row("Database", describeDatabase(cfg.DatabaseURL))
	row("Data", cfg.DataDir)
	row("Registration", onOff(cfg.RegistrationOpen, "open", "closed"))
	row("Admins", orNot(strings.Join(cfg.AdminUsers, ", "), "the account created on the first start"))
	row("Email", onOff(cfg.Mail.Host != "", fmt.Sprintf("%s:%d", cfg.Mail.Host, cfg.Mail.Port), "off"))
	row("Rate limits", onOff(cfg.RateLimits, "on", "off"))
	backups := "off"
	if cfg.Backups.Keep > 0 {
		backups = fmt.Sprintf("every %g hours, newest %d kept", cfg.Backups.Interval.Hours(), cfg.Backups.Keep)
	}
	row("Backups", backups)
	logs := orNot(cfg.Log.File, "stdout")
	if cfg.AnonymizeIPs {
		logs += ", IP addresses shortened"
	}
	row("Logs", fmt.Sprintf("%s (%s)", logs, cfg.Log.Level))
	b.WriteString("\n  Ready.\n\n")
	return b.String()
}

// listenAddress names the port; without a host the server listens on every interface.
func listenAddress(addr net.Addr) string {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	if ip := net.ParseIP(host); ip == nil || ip.IsUnspecified() {
		return fmt.Sprintf("port %s (all interfaces)", port)
	}
	return fmt.Sprintf("%s, port %s", host, port)
}

// describeDatabase leaves out the credentials of a PostgreSQL address.
func describeDatabase(databaseURL string) string {
	if path, ok := strings.CutPrefix(databaseURL, "sqlite:"); ok {
		return "SQLite " + path
	}
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "PostgreSQL"
	}
	return "PostgreSQL " + u.Host + u.Path
}

func onOff(on bool, yes, no string) string {
	if on {
		return yes
	}
	return no
}

func orNot(value, otherwise string) string {
	if value == "" {
		return otherwise
	}
	return value
}
