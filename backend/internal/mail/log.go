package mail

import (
	"context"
	"log/slog"
)

// LogSender writes messages to the log instead of sending them. It is the development
// fallback when no SMTP server is configured; the logs then contain the email links.
type LogSender struct {
	logger *slog.Logger
}

// NewLogSender creates a LogSender.
func NewLogSender(logger *slog.Logger) *LogSender {
	return &LogSender{logger: logger}
}

// Send logs m.
func (s *LogSender) Send(ctx context.Context, m Message) error {
	s.logger.InfoContext(ctx, "email not sent (no SMTP configured)", "to", m.To, "subject", m.Subject, "body", m.Body)
	return nil
}
