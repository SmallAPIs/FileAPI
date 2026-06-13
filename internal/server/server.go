// Package server wires HTTPS routing and middleware for the FileAPI agent.
package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/SmallAPIs/FileAPI/internal/config"
	"github.com/SmallAPIs/FileAPI/internal/filesystem"
	"github.com/SmallAPIs/FileAPI/internal/handlers"
	"github.com/SmallAPIs/FileAPI/internal/middleware"
	"github.com/SmallAPIs/FileAPI/internal/platform"
)

// Server is the HTTPS API server.
type Server struct {
	cfg    *config.Config
	http   *http.Server
	logger *slog.Logger
}

// New builds a configured HTTPS server.
func New(cfg *config.Config, logger *slog.Logger) (*Server, error) {
	fs, err := filesystem.NewService(cfg.AllowedRoots)
	if err != nil {
		return nil, err
	}

	files := handlers.NewFilesHandler(fs)
	system := handlers.NewSystemHandler(platform.New())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/v1/files", files.Read)
	mux.HandleFunc("POST /api/v1/files", files.Create)
	mux.HandleFunc("PATCH /api/v1/files", files.Edit)
	mux.HandleFunc("DELETE /api/v1/files", files.Delete)
	mux.HandleFunc("POST /api/v1/system/open-app", system.OpenApp)
	mux.HandleFunc("POST /api/v1/system/open-url", system.OpenURL)

	handler := middleware.CORS(cfg.AllowedOrigins)(
		middleware.Auth(
			recovery(logger)(
				logging(logger)(mux),
			),
		),
	)

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	return &Server{
		cfg:    cfg,
		http:   httpServer,
		logger: logger,
	}, nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	handlers.WriteOK(w, map[string]string{"status": "ok"})
}

// ListenAndServe starts the HTTPS server using configured certificate files.
func (s *Server) ListenAndServe() error {
	s.logger.Info("starting FileAPI agent",
		"listen", s.cfg.ListenAddr(),
		"api", s.cfg.BaseURL(),
		"config", s.cfg.ConfigPath,
		"cert", s.cfg.CertFile,
	)
	return s.http.ListenAndServeTLS(s.cfg.CertFile, s.cfg.KeyFile)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

func recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered", "error", fmt.Sprint(rec))
					handlers.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
