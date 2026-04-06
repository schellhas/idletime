package mail

import (
	"context"
	"log"
)

type LogMailer struct {
	logger *log.Logger
}

func NewLogMailer(logger *log.Logger) *LogMailer {
	if logger == nil {
		logger = log.Default()
	}

	return &LogMailer{logger: logger}
}

func (m *LogMailer) SendVerificationEmail(_ context.Context, toEmail, username, verificationURL string) error {
	m.logger.Printf(
		"verification email prepared for %s (%s): %s",
		username,
		toEmail,
		verificationURL,
	)
	return nil
}
