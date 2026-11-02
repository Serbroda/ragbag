package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/backup"
	"github.com/labstack/echo/v4"
)

// Running a public instance: the server backs itself up (BACKUP_KEEP files every BACKUP_INTERVAL_HOURS into
// BACKUP_DIR), answers health checks and can log IP addresses shortened.

// backupMu: one backup at a time, scheduled or by hand
var backupMu sync.Mutex

func (s *Server) backupDir() backup.Dir { return backup.Dir{Path: s.cfg.Backups.Dir} }

// makeBackup writes a backup and deletes the oldest beyond BACKUP_KEEP.
func (s *Server) makeBackup(ctx context.Context) (*backup.File, error) {
	backupMu.Lock()
	defer backupMu.Unlock()
	start := time.Now()
	file, err := s.backupDir().Create(ctx, s.store.DB(), s.store.Dialect, s.version)
	if err != nil {
		return nil, err
	}
	removed, err := s.backupDir().Prune(s.cfg.Backups.Keep)
	if err != nil {
		slog.Warn("deleting old backups failed", "error", err)
	}
	slog.Info("backup made", "file", file.Name, "bytes", file.Size, "ms", time.Since(start).Milliseconds(),
		"deleted_old", removed)
	return file, nil
}

// nextBackup is when the next scheduled backup is due: an interval after the newest one, so restarts do not
// add backups (or skip them).
func (s *Server) nextBackup() time.Time {
	files, err := s.backupDir().List()
	if err != nil || len(files) == 0 {
		return time.Now()
	}
	return files[0].CreatedAt.Add(s.cfg.Backups.Interval)
}

// ScheduledBackups runs until ctx ends (from main); nothing happens without BACKUP_KEEP.
func (s *Server) ScheduledBackups(ctx context.Context) {
	if s.cfg.Backups.Keep < 1 {
		return
	}
	slog.Info("scheduled backups", "dir", s.cfg.Backups.Dir, "keep", s.cfg.Backups.Keep,
		"every_hours", s.cfg.Backups.Interval.Hours())
	for {
		// A minute after starting at the earliest: the server comes up first
		wait := max(time.Until(s.nextBackup()), time.Minute)
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		if _, err := s.makeBackup(ctx); err != nil && ctx.Err() == nil {
			slog.Error("scheduled backup failed, trying again in an hour", "dir", s.cfg.Backups.Dir, "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Hour):
			}
		}
	}
}

func (s *Server) backupsView(c echo.Context) error {
	files, err := s.backupDir().List()
	if err != nil {
		return err
	}
	dto := api.BackupsDto{Enabled: s.cfg.Backups.Keep > 0, Keep: s.cfg.Backups.Keep,
		IntervalHours: int(s.cfg.Backups.Interval.Hours()), Dir: s.cfg.Backups.Dir, Files: []api.BackupFileDto{}}
	for _, file := range files {
		dto.Files = append(dto.Files, api.BackupFileDto{Name: file.Name, Size: file.Size, CreatedAt: file.CreatedAt})
	}
	if dto.Enabled {
		next := s.nextBackup()
		dto.NextAt = &next
	}
	return c.JSON(http.StatusOK, dto)
}

func (s *Server) GetBackups(c echo.Context) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	return s.backupsView(c)
}

func (s *Server) CreateBackup(c echo.Context) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	file, err := s.makeBackup(ctxOf(c))
	if err != nil {
		slog.Error("backup failed", "dir", s.cfg.Backups.Dir, "error", err)
		return fmt.Errorf("the backup could not be written to %s: %w", s.cfg.Backups.Dir, err)
	}
	audit(c, slog.LevelInfo, "backup made by hand", "file", file.Name)
	return s.backupsView(c)
}

func (s *Server) DownloadStoredBackup(c echo.Context, name string) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	path, err := s.backupDir().Open(name)
	if errors.Is(err, backup.ErrNoBackup) {
		return notFound("No such backup: %s", name)
	}
	if err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "stored backup downloaded", "file", name)
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename=%q`, name))
	c.Response().Header().Set(echo.HeaderContentType, "application/gzip")
	return c.File(path)
}

// Health is for monitoring (Uptime Kuma & co.) and the container's health check.
func (s *Server) Health(c echo.Context) error {
	ctx, cancel := context.WithTimeout(ctxOf(c), 3*time.Second)
	defer cancel()
	if err := s.store.DB().PingContext(ctx); err != nil {
		slog.Error("health check: the database does not answer", "error", err)
		return c.JSON(http.StatusServiceUnavailable, api.HealthDto{Status: api.Unavailable, Version: s.version})
	}
	return c.JSON(http.StatusOK, api.HealthDto{Status: api.Ok, Version: s.version})
}

// logIP is the client address as logged: with LOG_ANONYMIZE_IP an IPv4 address without its last byte, an IPv6
// one by its /48.
func (s *Server) logIP(c echo.Context) string {
	address := c.RealIP()
	if !s.cfg.AnonymizeIPs {
		return address
	}
	ip := net.ParseIP(address)
	switch {
	case ip == nil:
		return "-"
	case ip.To4() != nil:
		return ip.Mask(net.CIDRMask(24, 32)).String()
	default:
		return ip.Mask(net.CIDRMask(48, 128)).String()
	}
}
