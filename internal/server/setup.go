package server

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/auth"
	"github.com/Serbroda/ragbag/internal/backup"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// The first start of an empty instance creates the setup admin with a random password in a file next to the
// data (never in the log). Its first login replaces the password and offers to restore a backup, which moves an
// instance, e.g. from SQLite at home to PostgreSQL on a server.

const initialPasswordFile = "initial-admin-password"

func (s *Server) initialPasswordPath() string {
	return filepath.Join(s.cfg.DataDir, initialPasswordFile)
}

// Bootstrap creates the setup admin when there are no accounts yet.
func (s *Server) Bootstrap(ctx context.Context) error {
	count, err := s.store.CountUsers(ctx)
	if err != nil || count > 0 {
		return err
	}
	username := "admin"
	if len(s.cfg.AdminUsers) > 0 {
		username = strings.ToLower(s.cfg.AdminUsers[0])
	}
	password := rand.Text()
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	path := s.initialPasswordPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("directory for the initial admin password (DATA_DIR): %w", err)
	}
	// The file first: an account nobody can sign in to would block the setup for good
	if err := os.WriteFile(path, []byte(password+"\n"), 0o600); err != nil {
		return fmt.Errorf("writing the initial admin password (DATA_DIR): %w", err)
	}
	if _, err := s.store.CreateSetupAdmin(ctx, username, username+"@localhost", hash); err != nil {
		return err
	}
	slog.Warn("created the first account: sign in and choose your own password", "username", username,
		"password_file", path)
	return nil
}

func (s *Server) removeInitialPassword() {
	if err := os.Remove(s.initialPasswordPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("could not delete the initial admin password", "file", s.initialPasswordPath(), "error", err)
	}
}

// allowedDuringSetup: what the setup admin may do before replacing the initial password. Restoring a backup
// needs no own password first: it replaces the setup admin anyway (and only on an empty instance).
func allowedDuringSetup(method, requestPath string) bool {
	switch {
	case method == http.MethodGet && (requestPath == "/api/v1/users/me" || requestPath == "/api/v1/admin/instance"):
		return true
	case method == http.MethodPost && (requestPath == "/api/auth/setup" || requestPath == "/api/auth/password" ||
		requestPath == "/api/v1/admin/backup/restore"):
		return true
	}
	return false
}

// CompleteSetup replaces the initial password and sets the admin's email address.
func (s *Server) CompleteSetup(c echo.Context) error {
	if err := s.limitPasswordChange(c); err != nil {
		return err
	}
	var req api.CompleteSetupRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := s.store.UserByID(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	if !user.MustChangePassword {
		return conflict("The setup is already complete")
	}
	if !auth.CheckPassword(user.PasswordHash, req.CurrentPassword) {
		return badRequest("The initial password is wrong")
	}
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		return badRequest("%s", err.Error())
	}
	if req.NewPassword == req.CurrentPassword {
		return badRequest("Choose a new password")
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !strings.Contains(email, "@") || len(email) > 250 {
		return badRequest("Invalid email address")
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		if email != user.Email {
			if taken, err := tx.UsernameOrEmailTaken(ctxOf(c), email, email); err != nil || taken {
				if err == nil {
					err = conflict("Another account has this email address")
				}
				return err
			}
		}
		return tx.CompleteSetup(ctxOf(c), user.ID, email, hash)
	})
	if err != nil {
		return err
	}
	s.removeInitialPassword()
	if user, err = s.store.UserByID(ctxOf(c), user.ID); err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "setup complete: initial password replaced")
	return s.issueTokens(c, user)
}

func (s *Server) GetInstance(c echo.Context) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	empty, err := s.store.InstanceEmpty(ctxOf(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.InstanceDto{Empty: empty})
}

func (s *Server) DownloadBackup(c echo.Context) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "backup downloaded")
	res := c.Response()
	res.Header().Set(echo.HeaderContentType, "application/gzip")
	res.Header().Set(echo.HeaderContentDisposition,
		fmt.Sprintf(`attachment; filename="ragbag-backup-%s.json.gz"`, time.Now().Format("2006-01-02")))
	res.Header().Set("Cache-Control", "no-store")
	res.WriteHeader(http.StatusOK)
	// Errors after the first bytes can only end the download; the file is then cut off and refused on restore
	if err := backup.Write(ctxOf(c), s.store.DB(), s.store.Dialect, res, s.version); err != nil {
		slog.Error("backup failed", "error", err)
	}
	return nil
}

// RestoreBackup takes the upload into a temporary file first: it is read twice for older backups.
func (s *Server) RestoreBackup(c echo.Context, params api.RestoreBackupParams) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	file, err := os.CreateTemp("", "ragbag-backup-*.json.gz")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err := io.Copy(file, c.Request().Body); err != nil {
		return badRequest("Uploading the backup failed: %v", err)
	}
	if err := file.Close(); err != nil {
		return err
	}
	dryRun := params.DryRun != nil && *params.DryRun
	summary, err := backup.Restore(ctxOf(c), s.store.DB(), s.store.Dialect, file.Name(), dryRun)
	switch {
	case errors.Is(err, backup.ErrNotEmpty):
		return conflict("%s", err.Error())
	case err != nil:
		// The file's fault (or too new), not the server's
		slog.Warn("restoring a backup failed", "error", err)
		return badRequest("%s", err.Error())
	}
	if !dryRun {
		// Mainly the setup admin's file; the restored accounts sign in with their own passwords
		s.removeInitialPassword()
		audit(c, slog.LevelWarn, "backup restored: all data replaced", "backup_version", summary.AppVersion,
			"backup_created", summary.CreatedAt, "users", summary.Rows["users"])
	}
	usernames := []string{}
	for _, account := range summary.Accounts[:min(len(summary.Accounts), 20)] {
		usernames = append(usernames, account.Username)
	}
	return c.JSON(http.StatusOK, api.BackupSummaryDto{AppVersion: summary.AppVersion,
		SchemaVersion: summary.SchemaVersion, CreatedAt: summary.CreatedAt, Rows: summary.Rows,
		Admins: summary.Admins(s.cfg.AdminUsers), Usernames: usernames})
}
