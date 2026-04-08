package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"idletime/backend/internal/auth"
	"idletime/backend/internal/config"
	"idletime/backend/internal/database"
	httpapi "idletime/backend/internal/httpapi"
	"idletime/backend/internal/mail"
)

func main() {
	cfg := config.Load()

	dbPool, err := database.NewPostgresPool(context.Background(), cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer dbPool.Close()

	if err := database.ApplyMigrations(context.Background(), dbPool); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}

	mailer := mail.NewLogMailer(log.Default())
	authService := auth.NewService(dbPool, mailer, cfg.FrontendURL, cfg.SessionTTL())

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(cfg, dbPool, authService),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("idletime backend listening on http://localhost:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		_ = server.Close()
	}
}
