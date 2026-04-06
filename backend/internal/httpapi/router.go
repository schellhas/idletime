package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"idletime/backend/internal/auth"
	"idletime/backend/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(cfg config.Config, db *pgxpool.Pool, authService *auth.Service) http.Handler {
	mux := http.NewServeMux()
	authHandler := NewAuthHandler(cfg, authService)
	authHandler.RegisterRoutes(mux)

	categoryHandler := NewCategoryHandler(cfg, db, authService)
	categoryHandler.RegisterRoutes(mux)

	activityHandler := NewActivityHandler(cfg, db, authService)
	activityHandler.RegisterRoutes(mux)

	timeEntryHandler := NewTimeEntryHandler(cfg, db, authService)
	timeEntryHandler.RegisterRoutes(mux)

	recommendationHandler := NewRecommendationHandler(cfg, db, authService)
	recommendationHandler.RegisterRoutes(mux)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"service": "idletime-backend",
			"status":  "running",
		})
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "database_unavailable",
				"error":  err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ready",
		})
	})

	return withLogging(withCORS(cfg, mux))
}

func withCORS(cfg config.Config, next http.Handler) http.Handler {
	allowedOrigin := strings.TrimRight(cfg.FrontendURL, "/")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimRight(r.Header.Get("Origin"), "/")
		if origin != "" && origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,PUT,DELETE,OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started))
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json response: %v", err)
	}
}
