// Package mail composes and delivers transactional emails.
package mail

import (
	"context"
	"fmt"
)

// Message is a plain-text email.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender delivers messages.
type Sender interface {
	Send(ctx context.Context, m Message) error
}

// VerifyEmail asks a new user to confirm their address.
func VerifyEmail(to, username, link string) Message {
	return Message{
		To:      to,
		Subject: "Confirm your idletime account",
		Body: fmt.Sprintf(`Hi %s,

welcome to idletime! Please confirm your email address to activate your account:

%s

The link is valid for 24 hours. If you did not sign up, you can ignore this email.
`, username, link),
	}
}

// PasswordReset lets a user choose a new password.
func PasswordReset(to, username, link string) Message {
	return Message{
		To:      to,
		Subject: "Reset your idletime password",
		Body: fmt.Sprintf(`Hi %s,

we received a request to reset your password. Choose a new one here:

%s

The link is valid for 1 hour. If you did not request this, you can ignore this email;
your password stays unchanged.
`, username, link),
	}
}

// AccountExists tells the owner of an address that someone tried to register it again.
// Registration answers identically for known and unknown addresses, so this email is
// the only place the owner learns about it.
func AccountExists(to, loginLink, resetLink string) Message {
	return Message{
		To:      to,
		Subject: "You already have an idletime account",
		Body: fmt.Sprintf(`Hi,

someone tried to create an idletime account with this email address, but you already have one.

Log in: %s
Forgot your password? %s

If this was not you, you can ignore this email.
`, loginLink, resetLink),
	}
}
