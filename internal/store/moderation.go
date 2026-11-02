package store

import (
	"context"
	"strings"
	"time"
)

// --- Deleting accounts ---

// OwnedSpace is a space the user owns, with what deleting the account means for it.
type OwnedSpace struct {
	ID      string `db:"id"`
	Name    string `db:"name"`
	Owners  int    `db:"owners"`
	Members int    `db:"members"`
}

func (s *Store) OwnedSpaces(ctx context.Context, userID string) ([]OwnedSpace, error) {
	var spaces []OwnedSpace
	err := s.sel(ctx, &spaces, `SELECT s.id, s.name,
			(SELECT COUNT(*) FROM space_members o WHERE o.space_id = s.id AND o.role = 'OWNER') AS owners,
			(SELECT COUNT(*) FROM space_members a WHERE a.space_id = s.id) AS members
		FROM spaces s JOIN space_members m ON m.space_id = s.id
		WHERE m.user_id = ? AND m.role = 'OWNER' ORDER BY s.name`, userID)
	return spaces, err
}

// NextOwner is the member who takes over a space from its last owner: the highest role, then the longest
// membership.
func (s *Store) NextOwner(ctx context.Context, spaceID, leavingUserID string) (string, error) {
	var userID string
	return userID, s.get(ctx, &userID, `SELECT user_id FROM space_members WHERE space_id = ? AND user_id <> ?
		ORDER BY CASE role WHEN 'OWNER' THEN 0 WHEN 'ADMIN' THEN 1 WHEN 'CONTRIBUTOR' THEN 2 ELSE 3 END, created_at, user_id
		LIMIT 1`, spaceID, leavingUserID)
}

// AnonymizeUser deletes the personal data of an account that left all its spaces. The row stays, without
// name, email or password, so links it added to shared spaces keep their author; it can never sign in again.
func (s *Store) AnonymizeUser(ctx context.Context, user *User) error {
	statements := []string{
		`DELETE FROM invites WHERE created_by = ?`,
		`DELETE FROM collection_locks WHERE user_id = ?`,
		`DELETE FROM space_bars WHERE user_id = ?`,
		`DELETE FROM password_resets WHERE user_id = ?`,
		`DELETE FROM email_verifications WHERE user_id = ?`,
		`DELETE FROM dashboard_sections WHERE user_id = ?`,
		`DELETE FROM bookmark_favorites WHERE user_id = ?`,
		`DELETE FROM bookmark_reactions WHERE user_id = ?`,
		`DELETE FROM collection_visits WHERE user_id = ?`,
		`DELETE FROM collection_members WHERE user_id = ?`,
		// Images in shared spaces stay; they no longer count for anybody's quota
		`UPDATE images SET created_by = NULL WHERE created_by = ?`,
	}
	for _, statement := range statements {
		if _, err := s.exec(ctx, statement, user.ID); err != nil {
			return err
		}
	}
	if err := s.SetAvatar(ctx, user.ID, nil); err != nil {
		return err
	}
	if err := s.DeleteImage(ctx, user.AvatarImageID); err != nil {
		return err
	}
	// Names with characters registration refuses: nobody can take them, and they are unique by the id
	anonymous := "deleted-" + strings.ReplaceAll(user.ID, "-", "")[:12]
	at := now()
	_, err := s.exec(ctx, `UPDATE users SET username = ?, email = ?, password_hash = '', pin_hash = NULL,
		token_version = token_version + 1, start_space_id = NULL, setup_admin = ?, must_change_password = ?,
		blocked_at = NULL, blocked_reason = NULL, deleted_at = ?, updated_at = ? WHERE id = ?`,
		anonymous, user.ID+"@deleted.invalid", false, false, at, at, user.ID)
	return err
}

// --- Blocking accounts ---

// BlockUser also signs the account out everywhere.
func (s *Store) BlockUser(ctx context.Context, userID string, reason *string) error {
	at := now()
	_, err := s.exec(ctx, `UPDATE users SET blocked_at = ?, blocked_reason = ?, token_version = token_version + 1,
		updated_at = ? WHERE id = ?`, at, reason, at, userID)
	return err
}

func (s *Store) UnblockUser(ctx context.Context, userID string) error {
	_, err := s.exec(ctx, `UPDATE users SET blocked_at = NULL, blocked_reason = NULL, updated_at = ? WHERE id = ?`,
		now(), userID)
	return err
}

// --- Disabled public links ---

func (s *Store) DisableShare(ctx context.Context, collectionID string, reason *string) error {
	_, err := s.exec(ctx, `UPDATE collection_shares SET disabled_at = ?, disabled_reason = ? WHERE collection_id = ?`,
		now(), reason, collectionID)
	return err
}

func (s *Store) EnableShare(ctx context.Context, collectionID string) error {
	_, err := s.exec(ctx, `UPDATE collection_shares SET disabled_at = NULL, disabled_reason = NULL WHERE collection_id = ?`,
		collectionID)
	return err
}

// --- Abuse reports ---

type AbuseReport struct {
	ID           string  `db:"id"`
	ShareToken   string  `db:"share_token"`
	CollectionID *string `db:"collection_id"`
	// ViewedCollectionID is the sub-collection shown when reporting (or the shared one)
	ViewedCollectionID string     `db:"viewed_collection_id"`
	CollectionName     string     `db:"collection_name"`
	Category           string     `db:"category"`
	Details            string     `db:"details"`
	ReporterName       *string    `db:"reporter_name"`
	ReporterEmail      *string    `db:"reporter_email"`
	CreatedAt          time.Time  `db:"created_at"`
	ResolvedAt         *time.Time `db:"resolved_at"`
	ResolvedBy         *string    `db:"resolved_by"`
	Resolution         *string    `db:"resolution"`
	ResolutionNote     *string    `db:"resolution_note"`
}

// ReportView is a report with the current state of the reported link.
type ReportView struct {
	AbuseReport
	SpaceName       *string    `db:"space_name"`
	ShareCreatedBy  *string    `db:"share_created_by"`
	ShareCreator    *string    `db:"share_creator"`
	CurrentToken    *string    `db:"current_token"`
	ShareDisabledAt *time.Time `db:"share_disabled_at"`
	ResolvedByName  *string    `db:"resolved_by_name"`
}

const reportColumns = `r.id, r.share_token, r.collection_id, COALESCE(r.viewed_collection_id, '') AS viewed_collection_id, r.collection_name, r.category, r.details, r.reporter_name,
	r.reporter_email, r.created_at, r.resolved_at, r.resolved_by, r.resolution, r.resolution_note`

func (s *Store) CreateReport(ctx context.Context, report *AbuseReport) error {
	report.ID, report.CreatedAt = newID(), now()
	_, err := s.exec(ctx, `INSERT INTO abuse_reports (id, share_token, collection_id, viewed_collection_id, collection_name,
		category, details, reporter_name, reporter_email, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		report.ID, report.ShareToken, report.CollectionID, report.ViewedCollectionID, report.CollectionName, report.Category, report.Details,
		report.ReporterName, report.ReporterEmail, report.CreatedAt)
	return err
}

// Reports lists open reports first, then the resolved ones, newest first.
func (s *Store) Reports(ctx context.Context, limit int) ([]ReportView, error) {
	var reports []ReportView
	err := s.sel(ctx, &reports, `SELECT `+reportColumns+`, sp.name AS space_name, sh.created_by AS share_created_by,
			u.username AS share_creator, sh.token AS current_token, sh.disabled_at AS share_disabled_at,
			a.username AS resolved_by_name
		FROM abuse_reports r
		LEFT JOIN collections c ON c.id = r.collection_id
		LEFT JOIN spaces sp ON sp.id = c.space_id
		LEFT JOIN collection_shares sh ON sh.collection_id = r.collection_id
		LEFT JOIN users u ON u.id = sh.created_by
		LEFT JOIN users a ON a.id = r.resolved_by
		ORDER BY CASE WHEN r.resolved_at IS NULL THEN 0 ELSE 1 END, r.created_at DESC LIMIT ?`, limit)
	return reports, err
}

func (s *Store) ReportByID(ctx context.Context, id string) (*AbuseReport, error) {
	var report AbuseReport
	return &report, s.get(ctx, &report, `SELECT `+reportColumns+` FROM abuse_reports r WHERE r.id = ?`, id)
}

func (s *Store) CountOpenReports(ctx context.Context) (int, error) {
	var count int
	return count, s.get(ctx, &count, `SELECT COUNT(*) FROM abuse_reports WHERE resolved_at IS NULL`)
}

func (s *Store) ResolveReport(ctx context.Context, id, resolvedBy, resolution string, note *string) error {
	_, err := s.exec(ctx, `UPDATE abuse_reports SET resolved_at = ?, resolved_by = ?, resolution = ?, resolution_note = ?
		WHERE id = ?`, now(), resolvedBy, resolution, note, id)
	return err
}

// DeleteResolvedReports removes reports resolved before the time, with the reporters' contact data.
func (s *Store) DeleteResolvedReports(ctx context.Context, resolvedBefore time.Time) (int64, error) {
	return s.exec(ctx, `DELETE FROM abuse_reports WHERE resolved_at IS NOT NULL AND resolved_at < ?`, resolvedBefore.UTC())
}
