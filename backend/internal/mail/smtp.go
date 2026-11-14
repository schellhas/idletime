package mail

import (
	"context"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// SMTPConfig describes the outgoing mail server.
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// SMTPSender delivers messages through an SMTP server.
type SMTPSender struct {
	addr string
	auth smtp.Auth
	from mail.Address
}

// NewSMTPSender validates cfg and creates a sender. Authentication is used only when
// a username is configured.
func NewSMTPSender(cfg SMTPConfig) (*SMTPSender, error) {
	from, err := mail.ParseAddress(cfg.From)
	if err != nil {
		return nil, fmt.Errorf("parse SMTP from address: %w", err)
	}

	sender := &SMTPSender{addr: net.JoinHostPort(cfg.Host, cfg.Port), from: *from}
	if cfg.Username != "" {
		sender.auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	return sender, nil
}

// Send delivers m.
func (s *SMTPSender) Send(_ context.Context, m Message) error {
	if err := smtp.SendMail(s.addr, s.auth, s.from.Address, []string{m.To}, s.build(m)); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}

func (s *SMTPSender) build(m Message) []byte {
	headers := []string{
		"From: " + s.from.String(),
		"To: " + stripLineBreaks(m.To),
		"Subject: " + mime.QEncoding.Encode("utf-8", stripLineBreaks(m.Subject)),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Transfer-Encoding: 8bit",
	}
	body := strings.ReplaceAll(m.Body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\n", "\r\n")
	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body)
}

// stripLineBreaks prevents header injection through user-controlled values.
func stripLineBreaks(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}
