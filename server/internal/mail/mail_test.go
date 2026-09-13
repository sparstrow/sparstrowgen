package mail

import (
	"context"
	"io"
	"log/slog"
	"mime/quotedprintable"
	netmail "net/mail"
	"strings"
	"testing"
	"time"
)

// A link longer than a mail line must arrive in one piece. Quoted-printable
// soft-wraps it; decoding has to give back exactly the link that was written.
func TestComposeKeepsALongLinkIntact(t *testing.T) {
	from := &netmail.Address{Name: "sparstrowgen", Address: "no-reply@sparstrow.com"}
	link := "https://app.sparstrow.com/verify?token=" + strings.Repeat("Ab3_-", 20)
	raw := Compose(from, Message{
		To: "priya.nair@northwindlogistics.com", Subject: "Finish creating your sparstrowgen account",
		Body: "Finish creating your account:\n" + link + "\n",
	}, time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC), "abc123")

	msg, err := netmail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("the composed message does not parse: %v", err)
	}
	if got := msg.Header.Get("To"); got != "priya.nair@northwindlogistics.com" {
		t.Errorf("To = %q", got)
	}
	if got := msg.Header.Get("Message-ID"); got != "<abc123@sparstrow.com>" {
		t.Errorf("Message-ID = %q, want it on the sender's own domain", got)
	}
	body, err := io.ReadAll(quotedprintable.NewReader(msg.Body))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), link) {
		t.Errorf("the decoded body does not contain the link intact:\n%s", body)
	}
}

// A line break in a header would let whoever typed the address add headers of
// their own — a Bcc, say.
func TestALineBreakInAHeaderIsRefused(t *testing.T) {
	sender := Log{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	for _, m := range []Message{
		{To: "a@example.com\r\nBcc: everyone@example.com", Subject: "hi", Body: "x"},
		{To: "a@example.com", Subject: "hi\nBcc: everyone@example.com", Body: "x"},
		{To: "not an address", Subject: "hi", Body: "x"},
	} {
		if err := sender.Send(context.Background(), m); err == nil {
			t.Errorf("sent %+v; it should have been refused", m)
		}
	}
}

func TestSMTPValidateNamesWhatIsMissing(t *testing.T) {
	err := SMTP{Host: "smtp.hostinger.com", From: "sparstrowgen <no-reply@sparstrow.com>"}.Validate()
	if err == nil {
		t.Fatal("an SMTP configuration without credentials validated")
	}
	for _, name := range []string{"SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
	if err := (SMTP{Host: "h", Port: 465, Username: "u", Password: "p", From: "nope"}).Validate(); err == nil {
		t.Error("a MAIL_FROM that is not an address validated")
	}
}
