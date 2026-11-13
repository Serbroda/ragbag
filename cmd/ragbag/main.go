// Command ragbag runs the Ragbag server: REST API under /api, frontend under /app.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/Serbroda/ragbag/internal/db"
	"github.com/Serbroda/ragbag/internal/logging"
	"github.com/Serbroda/ragbag/internal/server"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/Serbroda/ragbag/ui"
)

// version is set at build time: -ldflags "-X main.version=1.2.3"
var version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	if len(os.Args) > 1 {
		// Commands log to stderr: a backup may go to stdout
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
		if err := command(os.Args[1], os.Args[2:]); err != nil {
			slog.Error(os.Args[1]+" failed", "error", err)
			os.Exit(1)
		}
		return
	}
	if err := run(); err != nil {
		slog.Error("ragbag stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	if err := config.LoadDotenv(); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Log.File != "" {
		// The last line on stdout (e.g. in the journal) says where to look
		slog.Info("logging to file", "file", cfg.Log.File, "level", cfg.Log.Level.String())
	}
	logs, err := logging.Setup(cfg.Log)
	if err != nil {
		return err
	}
	defer logs.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, dialect, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer database.Close()

	srv, err := server.New(cfg, store.New(database, dialect), version)
	if err != nil {
		return err
	}
	// The first start of an empty instance creates an admin with a random password in DATA_DIR, also in development
	if err := srv.Bootstrap(ctx); err != nil {
		return err
	}
	// What has been in the trash for 30 days is deleted for good, accounts never confirmed after 7 days; a new
	// log file every day, so LOG_MAX_AGE_DAYS holds
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		day := time.Now().YearDay()
		for {
			srv.PurgeOldTrash(ctx)
			srv.DeleteUnverifiedAccounts(ctx)
			srv.DeleteOldReports(ctx)
			if today := time.Now().YearDay(); today != day {
				day = today
				if err := logging.Rotate(); err != nil {
					slog.Error("rotating the log file failed", "error", err)
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	go srv.ScheduledBackups(ctx)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(ui.Frontend()),
		ReadHeaderTimeout: 10 * time.Second,
	}
	// Listening first, so the start message only comes once requests are answered (and a taken port fails here)
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return err
	}
	slog.Info("ragbag started", "version", version, "addr", listener.Addr().String(), "database", string(dialect),
		"log_level", cfg.Log.Level.String(), "registration", cfg.RegistrationOpen, "admins", len(cfg.AdminUsers),
		"private_hosts", cfg.AllowPrivateHosts)
	logging.Banner(cfg.Log, banner(cfg, version, listener.Addr()))
	errs := make(chan error, 1)
	go func() {
		errs <- httpServer.Serve(listener)
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
