package server

import (
	"context"
	"net/http"

	"github.com/Serbroda/ragbag/internal/store"
)

// Quotas (config.Quotas, with plans those of the user's plan; 0 = unlimited) are checked with what a request is
// about to add; checks run after the change in its transaction pass adding = 0 (imports, replaced images).

// quotaExceeded tells Free accounts when Pro has more (pro is that limit of Pro).
func (s *Server) quotaExceeded(free bool, limit, pro int, format string, args ...any) error {
	err := badRequest(format, args...).(*apiError)
	err.status, err.title = http.StatusForbidden, "Limit reached"
	if free && (pro <= 0 || pro > limit) {
		err.detail += ". Pro has more: account menu → Plan"
	}
	return err
}

func (s *Server) checkLinkQuota(ctx context.Context, tx *store.Store, userID string, adding int) error {
	quotas, free, err := s.userQuotas(ctx, tx, userID)
	if err != nil {
		return err
	}
	limit := quotas.LinksPerUser
	if limit <= 0 {
		return nil
	}
	count, err := tx.CountLinksBy(ctx, userID)
	if err != nil {
		return err
	}
	if count+adding > limit {
		return s.quotaExceeded(free, limit, s.cfg.Plans.Pro.LinksPerUser,
			"You can add at most %d links (the trash counts until it is emptied)", limit)
	}
	return nil
}

func (s *Server) checkCollectionQuota(ctx context.Context, tx *store.Store, spaceID string, adding int) error {
	quotas, free, err := s.spaceQuotas(ctx, tx, spaceID)
	if err != nil {
		return err
	}
	limit := quotas.CollectionsPerSpace
	if limit <= 0 {
		return nil
	}
	count, err := tx.CountCollections(ctx, spaceID)
	if err != nil {
		return err
	}
	if count+adding > limit {
		return s.quotaExceeded(free, limit, s.cfg.Plans.Pro.CollectionsPerSpace,
			"A space can have at most %d collections (the trash counts until it is emptied)", limit)
	}
	return nil
}

func (s *Server) checkSpaceQuota(ctx context.Context, tx *store.Store, userID string) error {
	quotas, free, err := s.userQuotas(ctx, tx, userID)
	if err != nil {
		return err
	}
	limit := quotas.SpacesPerUser
	if limit <= 0 {
		return nil
	}
	count, err := tx.CountOwnedSpaces(ctx, userID)
	if err != nil {
		return err
	}
	if count >= limit {
		return s.quotaExceeded(free, limit, s.cfg.Plans.Pro.SpacesPerUser, "You can own at most %d spaces", limit)
	}
	return nil
}

// checkUploadQuota runs after storing an image (and deleting the one it replaced).
func (s *Server) checkUploadQuota(ctx context.Context, tx *store.Store, userID string) error {
	quotas, free, err := s.userQuotas(ctx, tx, userID)
	if err != nil {
		return err
	}
	limit := quotas.UploadMBPerUser
	if limit <= 0 {
		return nil
	}
	total, err := tx.UploadedBytes(ctx, userID)
	if err != nil {
		return err
	}
	if total > int64(limit)<<20 {
		return s.quotaExceeded(free, limit, s.cfg.Plans.Pro.UploadMBPerUser,
			"Your images can take up at most %d MB; remove some preview images first", limit)
	}
	return nil
}
