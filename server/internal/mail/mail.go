// Package mail sends the few emails the product sends: a link to finish an
// account, a link to reset a password, and the notes around them.
//
// Plain text only. These are short, functional messages whose one job is a
// link, and plain text is what survives every mail client and spam filter
// without a template to maintain.
package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"mime/quotedprintable"
	"net"
	netmail "net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// Message is one email to one person.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender delivers a Message, or says why it could not.
type Sender interface {
	Send(ctx context.Context, m Message) error
}

// ---------------------------------------------------------------------------
// SMTP
// ---------------------------------------------------------------------------

// SMTP sends through an authenticated mail server — in production, the
// Hostinger mailbox on the product's own domain, so the mail is sent by the
// domain it claims to come from.
//
// Port 465 is TLS from the first byte. Any other port must offer STARTTLS, and
// sending is refused when it does not: these credentials would otherwise cross
// the network in the clear.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	// From is the sender as it appears to the recipient, e.g.
	// `sparstrowgen <no-reply@sparstrow.com>`.
	From string
}

// Validate reports what is missing or malformed, so the server can refuse to
// start rather than discovering it on the first person's sign-up.
func (s SMTP) Validate() error {
	var missing []string
	if s.Host == "" {
		missing = append(missing, "SMTP_HOST")
	}
	if s.Port <= 0 {
		missing = append(missing, "SMTP_PORT")
	}
	if s.Username == "" {
		missing = append(missing, "SMTP_USERNAME")
	}
	if s.Password == "" {
		missing = append(missing, "SMTP_PASSWORD")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing %s", strings.Join(missing, ", "))
	}
	if _, err := netmail.ParseAddress(s.From); err != nil {
		return fmt.Errorf("MAIL_FROM %q is not an address: %w", s.From, err)
	}
	return nil
}

func (s SMTP) Send(ctx context.Context, m Message) error {
	if err := headerSafe(m); err != nil {
		return err
	}
	from, err := netmail.ParseAddress(s.From)
	if err != nil {
		return fmt.Errorf("sender address: %w", err)
	}

	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	secure := &tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: 15 * time.Second}

	var conn net.Conn
	if s.Port == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, secure)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("connecting to the mail server %s: %w", addr, err)
	}
	// One deadline for the whole conversation with the server, so a mail server
	// that accepts the connection and then stalls cannot hold a request open.
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("greeting the mail server: %w", err)
	}
	defer func() { _ = client.Close() }()

	if s.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("the mail server does not offer STARTTLS; refusing to send credentials in the clear")
		}
		if err := client.StartTLS(secure); err != nil {
			return fmt.Errorf("starting TLS with the mail server: %w", err)
		}
	}
	if err := client.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
		return fmt.Errorf("signing in to the mail server: %w", err)
	}
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("mail server refused the sender: %w", err)
	}
	if err := client.Rcpt(m.To); err != nil {
		return fmt.Errorf("mail server refused the recipient: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail server refused the message: %w", err)
	}
	if _, err := w.Write(Compose(from, m, time.Now(), messageID())); err != nil {
		_ = w.Close()
		return fmt.Errorf("writing the message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mail server did not accept the message: %w", err)
	}
	return client.Quit()
}

// Compose builds the RFC 5322 message. Exported for its tests.
func Compose(from *netmail.Address, m Message, now time.Time, id string) []byte {
	var b bytes.Buffer
	header := func(name, value string) { fmt.Fprintf(&b, "%s: %s\r\n", name, value) }

	header("From", from.String())
	header("To", m.To)
	// Encoded, so a subject with a non-ASCII character in it is not sent as raw
	// bytes some servers reject.
	header("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	header("Date", now.Format(time.RFC1123Z))
	header("Message-ID", "<"+id+"@"+domainOf(from.Address)+">")
	header("MIME-Version", "1.0")
	header("Content-Type", `text/plain; charset="utf-8"`)
	header("Content-Transfer-Encoding", "quoted-printable")
	b.WriteString("\r\n")

	// Quoted-printable keeps long links intact: it soft-wraps lines at 76
	// characters in a way every client joins back up, where a raw 120-character
	// line can be broken by a relay.
	qp := quotedprintable.NewWriter(&b)
	_, _ = qp.Write([]byte(m.Body))
	_ = qp.Close()
	return b.Bytes()
}

// headerSafe refuses anything that could end a header early. The address and
// subject are built by this program, but the address started as something a
// person typed, and a newline in it would let them add headers — or recipients.
func headerSafe(m Message) error {
	if strings.ContainsAny(m.To, "\r\n") || strings.ContainsAny(m.Subject, "\r\n") {
		return errors.New("refusing to send: a header contains a line break")
	}
	if _, err := netmail.ParseAddress(m.To); err != nil {
		return fmt.Errorf("refusing to send to %q: %w", m.To, err)
	}
	return nil
}

func domainOf(address string) string {
	if at := strings.LastIndex(address, "@"); at >= 0 {
		return address[at+1:]
	}
	return "localhost"
}

func messageID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(raw)
}

// ---------------------------------------------------------------------------
// Log
// ---------------------------------------------------------------------------

// Log prints each message instead of sending it, links included. It exists for
// running the stack on a development machine without a mailbox, and is chosen
// only by setting MAIL_TRANSPORT=log explicitly — never as a fallback when
// SMTP is not configured. The server refuses it when session cookies are
// Secure, because a deployed server's log is not a place for working links.
type Log struct {
	Logger *slog.Logger
}

func (l Log) Send(_ context.Context, m Message) error {
	if err := headerSafe(m); err != nil {
		return err
	}
	l.Logger.Info("email NOT sent (MAIL_TRANSPORT=log)", "to", m.To, "subject", m.Subject)
	for _, line := range strings.Split(m.Body, "\n") {
		l.Logger.Info("  | " + line)
	}
	return nil
}
