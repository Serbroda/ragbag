package server

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

const inviteTargetSpace = "SPACE"

func inviteDto(invite *store.Invite) api.InviteDto {
	targetType := api.InviteDtoTargetType(invite.TargetType)
	role := api.InviteRoleEnum(invite.Role)
	var maxUses *int
	if invite.MaxUses >= 0 {
		maxUses = &invite.MaxUses
	}
	return api.InviteDto{
		Token: &invite.Token, TargetType: &targetType, TargetId: &invite.TargetID, Role: &role,
		ExpiresAt: invite.ExpiresAt, MaxUses: maxUses, UsedCount: &invite.UsedCount, CreatedAt: &invite.CreatedAt,
		CreatedBy: &invite.CreatedByName,
	}
}

func (s *Server) CreateInvite(c echo.Context) error {
	var req api.CreateInviteRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if string(req.TargetType) != inviteTargetSpace {
		return badRequest("Only space invites are supported")
	}
	role := RoleViewer
	if req.Role != nil {
		role = Role(*req.Role)
	}
	if role == RoleOwner || !validRole(role) {
		return badRequest("Ownership cannot be granted via invite")
	}
	if req.MaxUses != nil && *req.MaxUses < 1 {
		return badRequest("maxUses must be at least 1")
	}
	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
		return badRequest("expiresAt must be in the future")
	}
	inviterRole, err := s.store.MemberRole(ctxOf(c), req.TargetId, currentUser(c))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return forbidden("Not a member of the target space")
		}
		return err
	}
	// Only member managers may invite, and nobody may hand out a role above their own
	if !Role(inviterRole).Has(PermManageMembers) || !Role(inviterRole).AtLeast(role) {
		return forbidden("You do not have permission to create this invite")
	}

	invite := &store.Invite{
		Token: randomToken(), TargetType: inviteTargetSpace, TargetID: req.TargetId, Role: string(role),
		CreatedBy: currentUser(c), MaxUses: -1,
	}
	if req.MaxUses != nil {
		invite.MaxUses = *req.MaxUses
	}
	if req.ExpiresAt != nil {
		expiresAt := req.ExpiresAt.UTC()
		invite.ExpiresAt = &expiresAt
	}
	if err := s.store.CreateInvite(ctxOf(c), invite); err != nil {
		return err
	}
	created, err := s.store.InviteByToken(ctxOf(c), invite.Token)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, inviteDto(created))
}

func (s *Server) GetInvites(c echo.Context) error {
	invites, err := s.store.InvitesByCreator(ctxOf(c), currentUser(c))
	return respondInvites(c, invites, err)
}

func (s *Server) GetSpaceInvites(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermManageMembers); err != nil {
		return err
	}
	invites, err := s.store.InvitesForSpace(ctxOf(c), spaceID)
	return respondInvites(c, invites, err)
}

func respondInvites(c echo.Context, invites []store.Invite, err error) error {
	if err != nil {
		return err
	}
	result := make([]api.InviteDto, 0, len(invites))
	for i := range invites {
		result = append(result, inviteDto(&invites[i]))
	}
	return c.JSON(http.StatusOK, result)
}

// GetInvite previews an invite for the invite page: where it leads, by whom, whether it still works.
func (s *Server) GetInvite(c echo.Context, token string) error {
	invite, err := s.store.InviteByToken(ctxOf(c), token)
	if err != nil {
		return orNotFound(err, "Invite not found")
	}
	return s.invitePreview(c, invite)
}

func (s *Server) invitePreview(c echo.Context, invite *store.Invite) error {
	preview := api.InvitePreviewDto{
		TargetType: api.InvitePreviewDtoTargetType(invite.TargetType), TargetId: invite.TargetID,
		Role: ptr(api.InviteRoleEnum(invite.Role)), InvitedBy: &invite.CreatedByName, Valid: invite.Usable(),
	}
	if space, err := s.store.SpaceByID(ctxOf(c), invite.TargetID); err == nil {
		preview.TargetName = &space.Name
	}
	_, err := s.store.MemberRole(ctxOf(c), invite.TargetID, currentUser(c))
	preview.AlreadyMember = err == nil
	return c.JSON(http.StatusOK, preview)
}

func (s *Server) AcceptInvite(c echo.Context, token string) error {
	var invite *store.Invite
	err := s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		var err error
		if invite, err = tx.InviteByToken(ctxOf(c), token); err != nil {
			return orNotFound(err, "Invite not found")
		}
		if !invite.Usable() {
			return conflict("Invite is expired")
		}
		// Counted in the same statement that checks maxUses, so concurrent acceptances cannot exceed it
		used, err := tx.UseInvite(ctxOf(c), invite.ID)
		if err != nil {
			return err
		}
		if !used {
			return conflict("Invite is expired")
		}
		return joinSpace(ctxOf(c), tx, invite.TargetID, currentUser(c), Role(invite.Role))
	})
	if err != nil {
		return err
	}
	if invite, err = s.store.InviteByToken(ctxOf(c), token); err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "invite accepted", "space", invite.TargetID, "role", invite.Role)
	return s.invitePreview(c, invite)
}

func (s *Server) DeleteInvite(c echo.Context, token string) error {
	invite, err := s.store.InviteByToken(ctxOf(c), token)
	if err != nil {
		return orNotFound(err, "Invite not found")
	}
	creator := invite.CreatedBy == currentUser(c)
	role, err := s.store.MemberRole(ctxOf(c), invite.TargetID, currentUser(c))
	manager := err == nil && Role(role).Has(PermManageMembers)
	if !creator && !manager {
		return forbidden("Missing permission to revoke this invite")
	}
	if err := s.store.DeleteInvite(ctxOf(c), invite.ID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
