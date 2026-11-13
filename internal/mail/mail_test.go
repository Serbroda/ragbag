package mail

import (
	"bufio"
	"context"
	"io"
	"mime/quotedprintable"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeSMTP accepts one message without TLS (like a relay on localhost) and hands over what it got.
func fakeSMTP(t *testing.T) (port int, received chan string) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })
	received = make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		reply := func(line string) { _, _ = io.WriteString(conn, line+"\r\n") }
		reply("220 fake ESMTP")
		var transcript strings.Builder
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			transcript.WriteString(line)
			switch command := strings.ToUpper(strings.TrimSpace(line)); {
			case strings.HasPrefix(command, "EHLO"):
				reply("250-fake")
				reply("250 8BITMIME")
			case command == "DATA":
				reply("354 go ahead")
				for {
					data, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					if data == ".\r\n" {
						break
					}
					transcript.WriteString(data)
				}
				reply("250 queued")
			case command == "QUIT":
				reply("221 bye")
				received <- transcript.String()
				return
			default:
				reply("250 ok")
			}
		}
	}()
	return listener.Addr().(*net.TCPAddr).Port, received
}

func TestSendingWithoutTLS(t *testing.T) {
	port, received := fakeSMTP(t)
	sender, err := NewSMTP(Config{Host: "127.0.0.1", Port: port, From: "Ragbag <ragbag@example.com>", TLS: TLSNone})
	require.NoError(t, err)
	text := "Hi lena,\n\nconfirm here: https://ragbag.test/verify-email/abc\n\nÄrger? Ignore it."
	require.NoError(t, sender.Send(context.Background(), Message{To: "lena@example.com", Subject: "Bestätigen\r\nBcc: x@evil", Text: text}))

	transcript := <-received
	require.Contains(t, transcript, "MAIL FROM:<ragbag@example.com>")
	require.Contains(t, transcript, "RCPT TO:<lena@example.com>")
	message, _, _ := strings.Cut(transcript[strings.Index(transcript, "From: "):], "QUIT")
	head, body, _ := strings.Cut(message, "\r\n\r\n")
	require.Contains(t, head, `From: "Ragbag" <ragbag@example.com>`)
	require.Contains(t, head, "Subject: =?utf-8?q?")
	require.NotContains(t, head, "\r\nBcc:")
	decoded, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(body)))
	require.NoError(t, err)
	require.Equal(t, strings.ReplaceAll(text, "\n", "\r\n"), strings.TrimRight(string(decoded), "\r\n"))
}

func TestStartTLSIsRequiredUnlessTurnedOff(t *testing.T) {
	port, _ := fakeSMTP(t)
	sender, err := NewSMTP(Config{Host: "127.0.0.1", Port: port, From: "ragbag@example.com", TLS: TLSStartTLS})
	require.NoError(t, err)
	err = sender.Send(context.Background(), Message{To: "lena@example.com", Subject: "x", Text: "x"})
	require.ErrorContains(t, err, "STARTTLS")
}

func TestInvalidSettings(t *testing.T) {
	_, err := NewSMTP(Config{Host: "smtp.example.com", Port: 587, From: "not an address", TLS: TLSStartTLS})
	require.Error(t, err)
	_, err = NewSMTP(Config{Host: "smtp.example.com", Port: 587, From: "ragbag@example.com", TLS: "ssl"})
	require.Error(t, err)
}
