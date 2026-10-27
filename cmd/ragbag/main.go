// Command ragbag runs the Ragbag server: REST API under /api, frontend under /app.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/Serbroda/ragbag/internal/db"
	"github.com/Serbroda/ragbag/internal/server"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/Serbroda/ragbag/ui"
)

// version is set at build time: -ldflags "-X main.version=1.2.3"
var version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("ragbag stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, dialect, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer database.Close()

	srv := server.New(cfg, store.New(database, dialect), version)
	if cfg.Dev {
		slog.Warn("RAGBAG_DEV=true: demo account and generated JWT secret, not for production")
		if err := srv.SeedDemo(ctx); err != nil {
			return err
		}
	}
	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(ui.Frontend()),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errs := make(chan error, 1)
	go func() {
		slog.Info("ragbag started", "version", version, "addr", cfg.Addr, "database", string(dialect))
		errs <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errs:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
	return nil
}
