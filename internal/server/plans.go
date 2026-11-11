package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/config"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// Plans (PLANS_ENABLED): Free accounts get the QUOTA_* limits, Pro ones the QUOTA_PRO_* limits. Pro are instance
// admins and accounts an admin made Pro; nothing is sold. Without plans everybody gets QUOTA_*, as before.

// proSource says why an account is Pro, nil for Free and without plans.
func (s *Server) proSource(user *store.User) *api.ProSource {
	var source api.ProSource
	switch {
	case !s.cfg.Plans.Enabled:
		return nil
	case s.isAdmin(user):
		source = api.ProSourceADMIN
	case user.ProGrantedAt != nil:
		source = api.ProSourceGRANTED
	default:
		return nil
	}
	return &source
}

func (s *Server) planName(user *store.User) *api.PlanName {
	if !s.cfg.Plans.Enabled {
		return nil
	}
	plan := api.FREE
	if s.proSource(user) != nil {
		plan = api.PRO
	}
	return &plan
}

// quotasOf are the limits of the user's plan.
func (s *Server) quotasOf(user *store.User) config.Quotas {
	if s.proSource(user) != nil {
		return s.cfg.Plans.Pro
	}
	return s.cfg.Quotas
}

// userQuotas are the limits of the user and whether they are Free.
func (s *Server) userQuotas(ctx context.Context, tx *store.Store, userID string) (config.Quotas, bool, error) {
	if !s.cfg.Plans.Enabled {
		return s.cfg.Quotas, false, nil
	}
	user, err := tx.UserByID(ctx, userID)
	if err != nil {
		return config.Quotas{}, false, err
	}
	return s.quotasOf(user), s.proSource(user) == nil, nil
}

// spaceQuotas: a space has Pro limits when one of its owners is Pro.
func (s *Server) spaceQuotas(ctx context.Context, tx *store.Store, spaceID string) (config.Quotas, bool, error) {
	if !s.cfg.Plans.Enabled {
		return s.cfg.Quotas, false, nil
	}
	owners, err := tx.SpaceOwners(ctx, spaceID)
	if err != nil {
		return config.Quotas{}, false, err
	}
	for _, owner := range owners {
		if s.proSource(&owner) != nil {
			return s.cfg.Plans.Pro, false, nil
		}
	}
	return s.cfg.Quotas, true, nil
}

// --- The own plan ---

func (s *Server) requirePlans() error {
	if !s.cfg.Plans.Enabled {
		return notFound("This instance has no plans")
	}
	return nil
}

func (s *Server) GetPlan(c echo.Context) error {
	if err := s.requirePlans(); err != nil {
		return err
	}
	ctx := ctxOf(c)
	user, err := s.store.UserByID(ctx, currentUser(c))
	if err != nil {
		return err
	}
	links, err := s.store.CountLinksBy(ctx, user.ID)
	if err != nil {
		return err
	}
	spaces, err := s.store.CountOwnedSpaces(ctx, user.ID)
	if err != nil {
		return err
	}
	uploaded, err := s.store.UploadedBytes(ctx, user.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.PlanDto{Plan: *s.planName(user), ProSource: s.proSource(user),
		Usage: api.PlanUsageDto{Links: links, Spaces: spaces, UploadBytes: uploaded},
		Free:  quotasDto(s.cfg.Quotas), Pro: quotasDto(s.cfg.Plans.Pro), FreeApiTokens: s.cfg.Plans.FreeAPITokens})
}

func quotasDto(q config.Quotas) api.QuotasDto {
	return api.QuotasDto{LinksPerUser: q.LinksPerUser, SpacesPerUser: q.SpacesPerUser,
		CollectionsPerSpace: q.CollectionsPerSpace, UploadMbPerUser: q.UploadMBPerUser}
}

// --- Instance admins ---

func (s *Server) GrantPro(c echo.Context, userID api.ID) error {
	return s.setGrantedPro(c, userID, true)
}

func (s *Server) RevokePro(c echo.Context, userID api.ID) error {
	return s.setGrantedPro(c, userID, false)
}

func (s *Server) setGrantedPro(c echo.Context, userID string, granted bool) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	if err := s.requirePlans(); err != nil {
		return err
	}
	user, err := s.store.UserByID(ctxOf(c), userID)
	if err != nil || user.DeletedAt != nil {
		return notFound("User not found: %s", userID)
	}
	if err := s.store.GrantPro(ctxOf(c), user.ID, granted); err != nil {
		return err
	}
	if granted {
		audit(c, slog.LevelInfo, "pro granted", "granted_user", user.ID)
	} else {
		audit(c, slog.LevelInfo, "pro revoked", "revoked_user", user.ID)
	}
	return s.adminUserView(c, user.ID)
}
