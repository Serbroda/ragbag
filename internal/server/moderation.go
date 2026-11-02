package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/auth"
	"github.com/Serbroda/ragbag/internal/db"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// What a public instance needs: accounts delete themselves, instance admins block and delete accounts, and
// anybody can report a public link, which the admins disable or dismiss (notice and action, DSA art. 16/17).
// With email both sides learn the decision and its reasons.

const (
	// reportRetention: resolved reports, with the reporters' contact data, are deleted after that
	reportRetention = 180 * 24 * time.Hour
	maxReports      = 200
)

// --- Deleting accounts ---

func (s *Server) DeleteAccount(c echo.Context) error {
	if err := s.limitPasswordChange(c); err != nil {
		return err
	}
	var req api.DeleteAccountRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := s.store.UserByID(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		return badRequest("The password is wrong")
	}
	if err := s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		return s.removeAccount(c, tx, user, false)
	}); err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "account deleted by its owner")
	if s.mailEnabled() {
		s.sendMail(user, "Your Ragbag account was deleted", fmt.Sprintf(`Hi %s,

your Ragbag account and your personal data were deleted, as you asked. Spaces only you were a member of are
gone; links you added to spaces shared with others stay there without your name.
`, user.Username))
	}
	c.SetCookie(s.cookie(c, "", 0))
	return c.NoContent(http.StatusNoContent)
}

// DeleteUser is the same for instance admins, who cannot wait for the owner to hand over their spaces.
func (s *Server) DeleteUser(c echo.Context, userID api.ID) error {
	user, err := s.moderatedUser(c, userID)
	if err != nil {
		return err
	}
	if err := s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		return s.removeAccount(c, tx, user, true)
	}); err != nil {
		return err
	}
	audit(c, slog.LevelWarn, "account deleted by an instance admin", "deleted_user", user.ID)
	if s.mailEnabled() && user.EmailVerifiedAt != nil {
		s.sendMail(user, "Your Ragbag account was deleted", fmt.Sprintf(`Hi %s,

the administrators of %s deleted your Ragbag account and your personal data.
`, user.Username, s.cfg.PublicURL))
	}
	return c.NoContent(http.StatusNoContent)
}

// removeAccount deletes the spaces only the user is a member of and leaves the others. A space the user is the
// last owner of goes to the next member with handOver, otherwise it keeps the account from being deleted.
func (s *Server) removeAccount(c echo.Context, tx *store.Store, user *store.User, handOver bool) error {
	owned, err := tx.OwnedSpaces(ctxOf(c), user.ID)
	if err != nil {
		return err
	}
	var blocking []string
	for _, space := range owned {
		switch {
		case space.Members == 1:
			if err := deleteSpace(c, tx, space.ID); err != nil {
				return err
			}
		case space.Owners > 1:
		case handOver:
			next, err := tx.NextOwner(ctxOf(c), space.ID, user.ID)
			if err != nil {
				return err
			}
			if err := tx.SetMemberRole(ctxOf(c), space.ID, next, string(RoleOwner)); err != nil {
				return err
			}
			audit(c, slog.LevelInfo, "space handed over", "space", space.ID, "owner", next)
		default:
			blocking = append(blocking, `"`+space.Name+`"`)
		}
	}
	if len(blocking) > 0 {
		return conflict("You are the only owner of %s. Make another member an owner or delete the space first",
			strings.Join(blocking, ", "))
	}
	spaces, err := tx.SpacesForUser(ctxOf(c), user.ID)
	if err != nil {
		return err
	}
	for _, space := range spaces {
		if err := tx.RemoveMember(ctxOf(c), space.ID, user.ID); err != nil {
			return err
		}
	}
	return tx.AnonymizeUser(ctxOf(c), user)
}

func deleteSpace(c echo.Context, tx *store.Store, spaceID string) error {
	space, err := tx.SpaceByID(ctxOf(c), spaceID)
	if err != nil {
		return err
	}
	if err := tx.DeleteSpace(ctxOf(c), spaceID); err != nil {
		return err
	}
	return tx.DeleteImage(ctxOf(c), space.IconImageID)
}

// --- Blocking accounts ---

// moderatedUser is an account an instance admin may block or delete: not their own, not another admin's.
func (s *Server) moderatedUser(c echo.Context, userID string) (*store.User, error) {
	if err := s.requireInstanceAdmin(c); err != nil {
		return nil, err
	}
	user, err := s.store.UserByID(ctxOf(c), userID)
	if err != nil || user.DeletedAt != nil {
		return nil, notFound("User not found: %s", userID)
	}
	if user.ID == currentUser(c) {
		return nil, badRequest("You cannot do this to your own account")
	}
	if s.isAdmin(user) {
		return nil, badRequest("%s is an instance admin; remove them from ADMIN_USERS first", user.Username)
	}
	return user, nil
}

func (s *Server) BlockUser(c echo.Context, userID api.ID) error {
	user, err := s.moderatedUser(c, userID)
	if err != nil {
		return err
	}
	var req api.BlockUserRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	reason := trimmedOrNil(req.Reason)
	if reason != nil && len(*reason) > 1000 {
		return badRequest("The reason can have at most 1000 characters")
	}
	if err := s.store.BlockUser(ctxOf(c), user.ID, reason); err != nil {
		return err
	}
	audit(c, slog.LevelWarn, "account blocked", "blocked_user", user.ID)
	if s.mailEnabled() && user.EmailVerifiedAt != nil {
		s.sendMail(user, "Your Ragbag account was blocked", fmt.Sprintf(`Hi %s,

the administrators of %s blocked your Ragbag account. You cannot sign in anymore, and your public links no
longer work.

Reason: %s

If you think this is a mistake, answer this email or contact the administrators.
`, user.Username, s.cfg.PublicURL, valueOr(reason, "none given")))
	}
	return s.adminUserView(c, user.ID)
}

func (s *Server) UnblockUser(c echo.Context, userID api.ID) error {
	user, err := s.moderatedUser(c, userID)
	if err != nil {
		return err
	}
	if err := s.store.UnblockUser(ctxOf(c), user.ID); err != nil {
		return err
	}
	audit(c, slog.LevelWarn, "account unblocked", "unblocked_user", user.ID)
	if s.mailEnabled() && user.BlockedAt != nil && user.EmailVerifiedAt != nil {
		s.sendMail(user, "Your Ragbag account was unblocked", fmt.Sprintf(`Hi %s,

your Ragbag account at %s is unblocked: you can sign in again.
`, user.Username, s.cfg.PublicURL))
	}
	return s.adminUserView(c, user.ID)
}

func (s *Server) adminUserView(c echo.Context, userID string) error {
	users, err := s.store.AllUsers(ctxOf(c))
	if err != nil {
		return err
	}
	for _, user := range users {
		if user.ID == userID {
			return c.JSON(http.StatusOK, s.adminUserDto(user))
		}
	}
	return notFound("User not found: %s", userID)
}

func (s *Server) adminUserDto(user store.UserWithSpaces) api.AdminUserDto {
	createdAt, spaces := user.CreatedAt, user.Spaces
	return api.AdminUserDto{Id: user.ID, Username: user.Username, Email: user.Email, AvatarId: user.AvatarImageID,
		InstanceAdmin: s.isAdmin(&user.User), CreatedAt: &createdAt, Spaces: &spaces, BlockedAt: user.BlockedAt,
		BlockedReason: user.BlockedReason}
}

// blockedMessage is what a blocked account is told when signing in.
func blockedMessage(user *store.User) error {
	if user.BlockedReason != nil {
		return forbidden("This account was blocked by the administrators: %s", *user.BlockedReason)
	}
	return forbidden("This account was blocked by the administrators")
}

// --- Reports of public links ---

func (s *Server) ReportShare(c echo.Context, token string) error {
	var req api.CreateReportRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if !req.Category.Valid() {
		return badRequest("Unknown category: %s", req.Category)
	}
	details := strings.TrimSpace(req.Details)
	if len(details) < 10 || len(details) > 5000 {
		return badRequest("Describe the problem in 10 to 5000 characters")
	}
	if !req.GoodFaith {
		return badRequest("Please confirm that your report is accurate and complete")
	}
	name, email := trimmedOrNil(req.Name), trimmedOrNil(req.Email)
	if name != nil && len(*name) > 200 {
		return badRequest("The name can have at most 200 characters")
	}
	if email != nil && (!strings.Contains(*email, "@") || len(*email) > 250) {
		return badRequest("Invalid email address")
	}
	share, err := s.store.ShareByToken(ctxOf(c), token)
	if err != nil {
		return orNotFound(err, shareNotFound)
	}
	// Only what the link shows can be reported
	shared, err := s.sharedCollection(c, token, req.CollectionId)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(shared.Path))
	for _, ref := range shared.Path {
		names = append(names, ref.Name)
	}
	report := &store.AbuseReport{ShareToken: token, CollectionID: &share.CollectionID, ViewedCollectionID: shared.Id,
		CollectionName: strings.Join(names, " / "), Category: string(req.Category), Details: details,
		ReporterName: name, ReporterEmail: email}
	if err := s.store.CreateReport(ctxOf(c), report); err != nil {
		return err
	}
	audit(c, slog.LevelWarn, "public link reported", "report", report.ID, "collection", share.CollectionID,
		"category", report.Category)
	if s.mailEnabled() {
		s.notifyAdmins(ctxOf(c), report)
		if email != nil {
			s.deliver(*email, "reporter", "Your report to Ragbag", fmt.Sprintf(`Hello,

thank you for your report of %s/app/share/%s ("%s"). The administrators will look at it and let you know
what they decided.
`, s.cfg.PublicURL, token, report.CollectionName))
		}
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) notifyAdmins(ctx context.Context, report *store.AbuseReport) {
	users, err := s.store.AllUsers(ctx)
	if err != nil {
		slog.Error("finding the admins for a report failed", "error", err)
		return
	}
	for _, user := range users {
		if !s.isAdmin(&user.User) || user.EmailVerifiedAt == nil || strings.HasSuffix(user.Email, "@localhost") {
			continue
		}
		s.sendMail(&user.User, "New report of a public link", fmt.Sprintf(`Hi %s,

somebody reported the public link %s/app/share/%s ("%s") as %s:

%s

Decide in Ragbag: %s/app/admin/reports
`, user.Username, s.cfg.PublicURL, report.ShareToken, report.CollectionName, report.Category, report.Details, s.cfg.PublicURL))
	}
}

func (s *Server) GetReports(c echo.Context) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	reports, err := s.store.Reports(ctxOf(c), maxReports)
	if err != nil {
		return err
	}
	result := make([]api.AbuseReportDto, 0, len(reports))
	for _, report := range reports {
		result = append(result, reportDto(report))
	}
	return c.JSON(http.StatusOK, result)
}

func reportDto(r store.ReportView) api.AbuseReportDto {
	dto := api.AbuseReportDto{Id: r.ID, Category: api.ReportCategory(r.Category), Details: r.Details,
		ReporterName: r.ReporterName, ReporterEmail: r.ReporterEmail, CreatedAt: r.CreatedAt,
		CollectionId: r.CollectionID, CollectionName: r.CollectionName, SpaceName: r.SpaceName,
		ShareToken: r.ShareToken, CurrentToken: r.CurrentToken, SharedBy: r.ShareCreator, SharedById: r.ShareCreatedBy,
		LinkDisabled: r.ShareDisabledAt != nil, ResolvedAt: r.ResolvedAt, ResolvedBy: r.ResolvedByName,
		ResolutionNote: r.ResolutionNote}
	if r.ViewedCollectionID != "" {
		dto.ViewedCollectionId = &r.ViewedCollectionID
	}
	if r.Resolution != nil {
		resolution := api.AbuseReportDtoResolution(*r.Resolution)
		dto.Resolution = &resolution
	}
	return dto
}

func (s *Server) ResolveReport(c echo.Context, reportID api.ID) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	var req api.ResolveReportRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if !req.Action.Valid() {
		return badRequest("Unknown action: %s", req.Action)
	}
	note := trimmedOrNil(req.Note)
	if note != nil && len(*note) > 2000 {
		return badRequest("The note can have at most 2000 characters")
	}
	report, err := s.store.ReportByID(ctxOf(c), reportID)
	if err != nil {
		return orNotFound(err, "Report not found: %s", reportID)
	}
	var share *store.Share
	if req.Action != api.ResolveReportRequestActionDISMISS {
		if report.CollectionID != nil {
			share, err = s.store.ShareOf(ctxOf(c), *report.CollectionID)
		}
		if report.CollectionID == nil || errors.Is(err, store.ErrNotFound) {
			return badRequest("The collection has no public link anymore")
		}
		if err != nil {
			return err
		}
	}
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		switch req.Action {
		case api.ResolveReportRequestActionDISABLELINK:
			if err := tx.DisableShare(ctxOf(c), share.CollectionID, note); err != nil {
				return err
			}
		case api.ResolveReportRequestActionENABLELINK:
			if err := tx.EnableShare(ctxOf(c), share.CollectionID); err != nil {
				return err
			}
		}
		return tx.ResolveReport(ctxOf(c), report.ID, currentUser(c), string(req.Action), note)
	})
	if err != nil {
		return err
	}
	audit(c, slog.LevelWarn, "report resolved", "report", report.ID, "action", req.Action)
	if s.mailEnabled() {
		s.mailDecision(ctxOf(c), report, share, req.Action, note)
	}
	reports, err := s.store.Reports(ctxOf(c), maxReports)
	if err != nil {
		return err
	}
	for _, view := range reports {
		if view.ID == report.ID {
			return c.JSON(http.StatusOK, reportDto(view))
		}
	}
	return notFound("Report not found: %s", reportID)
}

// mailDecision tells the reporter and, when the link changed, the one who shared it (statement of reasons).
func (s *Server) mailDecision(ctx context.Context, report *store.AbuseReport, share *store.Share,
	action api.ResolveReportRequestAction, note *string) {
	link := fmt.Sprintf(`%s/app/share/%s ("%s")`, s.cfg.PublicURL, report.ShareToken, report.CollectionName)
	reasons := valueOr(note, "none given")
	if report.ReporterEmail != nil && action != api.ResolveReportRequestActionENABLELINK {
		decision := "The administrators disabled the link."
		if action == api.ResolveReportRequestActionDISMISS {
			decision = "The administrators looked at the link and decided to leave it online."
		}
		s.deliver(*report.ReporterEmail, "reporter", "The decision on your report", fmt.Sprintf(`Hello,

thank you for your report of %s.

%s

Reasons: %s
`, link, decision, reasons))
	}
	if share == nil {
		return
	}
	creator, err := s.store.UserByID(ctx, share.CreatedBy)
	if err != nil || creator.DeletedAt != nil || creator.EmailVerifiedAt == nil {
		return
	}
	switch action {
	case api.ResolveReportRequestActionDISABLELINK:
		s.sendMail(creator, "Your public link was disabled", fmt.Sprintf(`Hi %s,

after a report, the administrators of %s disabled the public link %s, which you created. The
collection itself is unchanged, but it cannot be shared publicly anymore.

Reasons: %s

If you think this is a mistake, answer this email or contact the administrators.
`, creator.Username, s.cfg.PublicURL, link, reasons))
	case api.ResolveReportRequestActionENABLELINK:
		s.sendMail(creator, "Your public link works again", fmt.Sprintf(`Hi %s,

the administrators enabled your public link %s again.
`, creator.Username, link))
	}
}

// DeleteOldReports removes resolved reports after reportRetention (hourly job).
func (s *Server) DeleteOldReports(ctx context.Context) {
	deleted, err := s.store.DeleteResolvedReports(ctx, time.Now().Add(-reportRetention))
	if err != nil {
		slog.Error("deleting old reports failed", "error", err)
		return
	}
	if deleted > 0 {
		slog.Info("deleted old reports", "reports", deleted)
	}
}

func trimmedOrNil(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func valueOr(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}

// --- Admin page ---

func (s *Server) GetAdminOverview(c echo.Context) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	stats, err := s.store.Stats(ctxOf(c))
	if err != nil {
		return err
	}
	version, err := db.SchemaVersion(ctxOf(c), s.store.DB(), s.store.Dialect)
	if err != nil {
		return err
	}
	admins := s.cfg.AdminUsers
	if admins == nil {
		admins = []string{}
	}
	quotas := s.cfg.Quotas
	return c.JSON(http.StatusOK, api.AdminOverviewDto{
		Version: s.version, Database: api.AdminOverviewDtoDatabase(s.store.Dialect), SchemaVersion: version,
		Users: stats.Users, BlockedUsers: stats.BlockedUsers, UnconfirmedUsers: stats.UnconfirmedUsers,
		Spaces: stats.Spaces, Collections: stats.Collections, Bookmarks: stats.Bookmarks,
		PublicLinks: stats.PublicLinks, ImageBytes: stats.ImageBytes, OpenReports: stats.OpenReports,
		Settings: api.AdminSettingsDto{PublicUrl: s.cfg.PublicURL, RegistrationOpen: s.cfg.RegistrationOpen,
			Email: s.mailEnabled(), RateLimits: s.cfg.RateLimits, AllowPrivateHosts: s.cfg.AllowPrivateHosts,
			AdminUsers: admins, QuotaLinksPerUser: quotas.LinksPerUser, QuotaSpacesPerUser: quotas.SpacesPerUser,
			QuotaCollectionsPerSpace: quotas.CollectionsPerSpace, QuotaUploadMbPerUser: quotas.UploadMBPerUser},
	})
}
