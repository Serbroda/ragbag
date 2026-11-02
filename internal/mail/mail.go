// Package mail sends the few emails Ragbag needs (confirming an address, resetting a password) over SMTP.
package mail

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
	"strconv"
	"strings"
	"time"
)

type Message struct {
	To, Subject, Text string
}

// Sender delivers messages; tests replace the SMTP one.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// TLS modes: STARTTLS on the submission port (587), implicit TLS (465) or none (a relay on localhost).
const (
	TLSStartTLS = "starttls"
	TLSImplicit = "tls"
	TLSNone     = "none"
)

type Config struct {
	Host               string
	Port               int
	Username, Password string
	// From is the sender, e.g. "Ragbag <ragbag@example.com>"
	From string
	TLS  string
}

type SMTP struct {
	cfg  Config
	from *mail.Address
}

func NewSMTP(cfg Config) (*SMTP, error) {
	from, err := mail.ParseAddress(cfg.From)
	if err != nil {
		return nil, fmt.Errorf("SMTP_FROM %q is no valid address: %w", cfg.From, err)
	}
	switch cfg.TLS {
	case TLSStartTLS, TLSImplicit, TLSNone:
	default:
		return nil, fmt.Errorf("SMTP_TLS must be starttls, tls or none, not %q", cfg.TLS)
	}
	return &SMTP{cfg: cfg, from: from}, nil
}

const timeout = 30 * time.Second

func (s *SMTP) Send(ctx context.Context, msg Message) error {
	to, err := mail.ParseAddress(msg.To)
	if err != nil {
		return fmt.Errorf("invalid recipient: %w", err)
	}
	body, err := s.compose(to, msg)
	if err != nil {
		return err
	}

	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	deadline, _ := ctx.Deadline()
	_ = conn.SetDeadline(deadline)
	tlsConfig := &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}
	if s.cfg.TLS == TLSImplicit {
		conn = tls.Client(conn, tlsConfig)
	}
	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()
	if s.cfg.TLS == TLSStartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("the SMTP server offers no STARTTLS (set SMTP_TLS=tls for port 465)")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if s.cfg.Username != "" {
		// PlainAuth refuses to send the password without TLS, except to localhost
		if err := client.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			return err
		}
	}
	if err := client.Mail(s.from.Address); err != nil {
		return err
	}
	if err := client.Rcpt(to.Address); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(body); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func (s *SMTP) compose(to *mail.Address, msg Message) ([]byte, error) {
	var buf bytes.Buffer
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	domain := s.from.Address[strings.LastIndex(s.from.Address, "@")+1:]
	headers := []string{
		"From: " + s.from.String(),
		"To: " + to.String(),
		// Q-encoded, so a subject can never break into further headers
		"Subject: " + mime.QEncoding.Encode("utf-8", msg.Subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"Message-ID: <" + hex.EncodeToString(id) + "@" + domain + ">",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Transfer-Encoding: quoted-printable",
		"Auto-Submitted: auto-generated",
	}
	buf.WriteString(strings.Join(headers, "\r\n") + "\r\n\r\n")
	writer := quotedprintable.NewWriter(&buf)
	if _, err := writer.Write([]byte(strings.ReplaceAll(msg.Text, "\n", "\r\n"))); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
