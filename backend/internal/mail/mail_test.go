package mail

import (
	"strings"
	"testing"
)

func TestSMTPSenderBuildPreventsHeaderInjection(t *testing.T) {
	sender, err := NewSMTPSender(SMTPConfig{Host: "localhost", Port: "25", From: "idletime <no-reply@example.com>"})
	if err != nil {
		t.Fatalf("NewSMTPSender() error = %v", err)
	}

	raw := string(sender.build(Message{
		To:      "victim@example.com\r\nBcc: attacker@example.com",
		Subject: "Hello\r\nBcc: attacker@example.com",
		Body:    "line one\nline two",
	}))

	headers, body, found := strings.Cut(raw, "\r\n\r\n")
	if !found {
		t.Fatal("message has no header/body separator")
	}
	for _, line := range strings.Split(headers, "\r\n") {
		if strings.HasPrefix(line, "Bcc:") {
			t.Errorf("injected header found: %q", line)
		}
	}
	if body != "line one\r\nline two" {
		t.Errorf("body = %q, want CRLF line endings", body)
	}
}

func TestNewSMTPSenderRejectsInvalidFrom(t *testing.T) {
	if _, err := NewSMTPSender(SMTPConfig{Host: "h", Port: "25", From: "not an address"}); err == nil {
		t.Fatal("expected an error for an invalid from address")
	}
}

func TestTemplatesContainLinks(t *testing.T) {
	if m := VerifyEmail("a@example.com", "alice", "https://x/verify?token=1"); !strings.Contains(m.Body, "https://x/verify?token=1") {
		t.Error("verification email lacks the link")
	}
	if m := PasswordReset("a@example.com", "alice", "https://x/reset?token=2"); !strings.Contains(m.Body, "https://x/reset?token=2") {
		t.Error("reset email lacks the link")
	}
}
