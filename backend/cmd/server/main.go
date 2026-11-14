// Command server runs the idletime HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/schellhas/idletime/backend/internal/account"
	"github.com/schellhas/idletime/backend/internal/auth"
	"github.com/schellhas/idletime/backend/internal/category"
	"github.com/schellhas/idletime/backend/internal/config"
	"github.com/schellhas/idletime/backend/internal/database"
	"github.com/schellhas/idletime/backend/internal/mail"
	"github.com/schellhas/idletime/backend/internal/ratelimit"
	"github.com/schellhas/idletime/backend/internal/server"
	"github.com/schellhas/idletime/backend/internal/user"
)

const (
	shutdownTimeout = 10 * time.Second

	// Sensitive auth endpoints allow this many requests per client and window.
	authRateLimit  = 10
	authRateWindow = time.Minute
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server terminated", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := database.Migrate(ctx, cfg.DatabaseURL); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	mailer, err := newMailer(cfg, logger)
	if err != nil {
		return err
	}

	users := user.NewRepository(pool)
	sessions := auth.NewSessionStore(pool)
	authService := auth.NewService(users, sessions, auth.NewTokenStore(pool), mailer, cfg.AppBaseURL)
	categoryService := category.NewService(category.NewPostgresRepository(pool))

	guard := auth.Guard(authService, cfg.CookieSecure)
	clientIP := func(r *http.Request) string { return ratelimit.ClientIP(r, cfg.TrustProxyHeaders) }
	limit := ratelimit.Middleware(ratelimit.New(authRateLimit, authRateWindow), clientIP)

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: server.NewRouter(server.Dependencies{
			Logger:        logger,
			Pinger:        pool,
			AllowedOrigin: cfg.AppOrigin(),
			Routes: []server.RouteRegistrar{
				auth.NewHandler(authService, limit, clientIP, cfg.CookieSecure),
				account.NewHandler(account.NewService(users, sessions, categoryService), guard, cfg.CookieSecure),
				category.NewHandler(categoryService, guard),
			},
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", cfg.HTTPAddr)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	return nil
}

// newMailer sends real emails when SMTP is configured and otherwise logs them.
func newMailer(cfg config.Config, logger *slog.Logger) (mail.Sender, error) {
	if cfg.SMTP.Host == "" {
		logger.Warn("SMTP_HOST is not set: emails are logged instead of sent")
		return mail.NewLogSender(logger), nil
	}
	sender, err := mail.NewSMTPSender(cfg.SMTP)
	if err != nil {
		return nil, fmt.Errorf("configure SMTP: %w", err)
	}
	return sender, nil
}
