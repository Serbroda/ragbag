package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Serbroda/ragbag/internal/backup"
	"github.com/Serbroda/ragbag/internal/config"
	"github.com/Serbroda/ragbag/internal/db"
)

const usage = `Usage:
  ragbag                          run the server
  ragbag backup [file]            write a backup of DATABASE_URL to file (default: stdout)
  ragbag restore [-dry-run] file  restore a backup into the empty database at DATABASE_URL
  ragbag version                  print the version

Both commands read the same environment (and .env/.env.local) as the server; the server may keep running
during a backup.
`

func command(name string, args []string) error {
	switch name {
	case "backup":
		return backupCommand(args)
	case "restore":
		return restoreCommand(args)
	case "version", "--version", "-v":
		fmt.Println(version)
		return nil
	case "help", "--help", "-h":
		fmt.Print(usage)
		return nil
	}
	fmt.Fprint(os.Stderr, usage)
	return fmt.Errorf("unknown command %q", name)
}

// openDatabase needs only DATABASE_URL, not the server's whole configuration (JWT_SECRET & co.).
func openDatabase(ctx context.Context) (*db.Handle, error) {
	if err := config.LoadDotenv(); err != nil {
		return nil, err
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "sqlite:data/ragbag.db"
	}
	database, dialect, err := db.Open(ctx, url)
	if err != nil {
		return nil, err
	}
	return &db.Handle{DB: database, Dialect: dialect, URL: url}, nil
}

func backupCommand(args []string) error {
	if len(args) > 1 {
		return errors.New("usage: ragbag backup [file]")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	database, err := openDatabase(ctx)
	if err != nil {
		return err
	}
	defer database.DB.Close()
	var out io.Writer = os.Stdout
	if len(args) == 1 && args[0] != "-" {
		// Holds password hashes: readable for the owner only
		file, err := os.OpenFile(args[0], os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		defer file.Close()
		out = file
	}
	if err := backup.Write(ctx, database.DB, database.Dialect, out, version); err != nil {
		return err
	}
	if len(args) == 1 && args[0] != "-" {
		slog.Info("backup written", "file", args[0])
	}
	return nil
}

func restoreCommand(args []string) error {
	flags := flag.NewFlagSet("restore", flag.ContinueOnError)
	dryRun := flags.Bool("dry-run", false, "only check the backup")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: ragbag restore [-dry-run] file")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	database, err := openDatabase(ctx)
	if err != nil {
		return err
	}
	defer database.DB.Close()
	summary, err := backup.Restore(ctx, database.DB, database.Dialect, flags.Arg(0), *dryRun)
	if err != nil {
		return err
	}
	verb := "checked (nothing changed)"
	if !*dryRun {
		verb = "restored"
		// The setup admin was replaced; its password file is of no use anymore
		_ = os.Remove(filepath.Join(config.DataDir(database.URL), "initial-admin-password"))
	}
	slog.Info("backup "+verb, "made_by", summary.AppVersion, "created", summary.CreatedAt, "users", summary.Rows["users"],
		"spaces", summary.Rows["spaces"], "links", summary.Rows["bookmarks"], "images", summary.Rows["images"])
	return nil
}
