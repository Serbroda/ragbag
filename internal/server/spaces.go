package server

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

func spaceDto(space *store.Space, role Role) api.SpaceDto {
	apiRole := api.SpaceMemberRoleEnum(role)
	permissions := []api.SpacePermissionEnum{}
	for _, p := range allPermissions {
		if role.Has(p) {
			permissions = append(permissions, api.SpacePermissionEnum(p))
		}
	}
	joinRole := api.JoinRoleEnum(space.JoinRole)
	return api.SpaceDto{Id: space.ID, Name: space.Name, Description: space.Description, IconId: space.IconImageID,
		Role: &apiRole, Permissions: &permissions, Discoverable: &space.Discoverable, JoinRole: &joinRole}
}

// maxDiscoverResults keeps the discover list short; the search narrows it down.
const maxDiscoverResults = 100

func memberDto(member store.Member) api.SpaceMemberDto {
	return api.SpaceMemberDto{UserId: member.UserID, Username: member.Username, AvatarId: member.AvatarImageID,
		Role: api.SpaceMemberRoleEnum(member.Role)}
}

func requireName(name, what string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", badRequest("%s name must not be empty", what)
	}
	if len([]rune(trimmed)) > 80 {
		return "", badRequest("%s name must be at most 80 characters", what)
	}
	return trimmed, nil
}

func blankToNil(value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func (s *Server) GetSpaces(c echo.Context) error {
	spaces, err := s.store.SpacesForUser(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	// Owned spaces first, like Discord lists your own servers on top
	sort.SliceStable(spaces, func(i, j int) bool {
		ri, rj := Role(spaces[i].Role).rank(), Role(spaces[j].Role).rank()
		if ri != rj {
			return ri < rj
		}
		return strings.ToLower(spaces[i].Name) < strings.ToLower(spaces[j].Name)
	})
	result := make([]api.SpaceDto, 0, len(spaces))
	for _, space := range spaces {
		result = append(result, spaceDto(&space.Space, Role(space.Role)))
	}
	return c.JSON(http.StatusOK, result)
}

func (s *Server) CreateSpace(c echo.Context) error {
	var req api.CreateSpaceDto
	if err := bind(c, &req); err != nil {
		return err
	}
	name, err := requireName(req.Name, "Space")
	if err != nil {
		return err
	}
	var space *store.Space
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		if space, err = tx.CreateSpace(ctxOf(c), name, blankToNil(req.Description), currentUser(c)); err != nil {
			return err
		}
		return tx.AddMember(ctxOf(c), space.ID, currentUser(c), string(RoleOwner))
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, spaceDto(space, RoleOwner))
}

func (s *Server) GetSpace(c echo.Context, spaceID api.ID) error {
	role, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead)
	if err != nil {
		return err
	}
	return s.spaceView(c, spaceID, role)
}

func (s *Server) spaceView(c echo.Context, spaceID string, role Role) error {
	space, err := s.store.SpaceByID(ctxOf(c), spaceID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, spaceDto(space, role))
}

func (s *Server) UpdateSpace(c echo.Context, spaceID api.ID) error {
	role, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermEditSpace)
	if err != nil {
		return err
	}
	var req api.UpdateSpaceDto
	if err := bind(c, &req); err != nil {
		return err
	}
	name, err := requireName(req.Name, "Space")
	if err != nil {
		return err
	}
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		space, err := tx.SpaceByID(ctxOf(c), spaceID)
		if err != nil {
			return err
		}
		discoverable, joinRole := space.Discoverable, space.JoinRole
		if req.Discoverable != nil {
			discoverable = *req.Discoverable
		}
		if req.JoinRole != nil {
			if !req.JoinRole.Valid() {
				return badRequest("Unknown join role: %s", *req.JoinRole)
			}
			joinRole = string(*req.JoinRole)
		}
		if discoverable != space.Discoverable || joinRole != space.JoinRole {
			// Letting everybody in is a membership decision
			if !role.Has(PermManageMembers) {
				return forbidden("Missing permission %s", PermManageMembers)
			}
			if err := tx.SetSpaceDiscoverable(ctxOf(c), spaceID, discoverable, joinRole); err != nil {
				return err
			}
		}
		return tx.UpdateSpace(ctxOf(c), spaceID, name, blankToNil(req.Description))
	})
	if err != nil {
		return err
	}
	return s.spaceView(c, spaceID, role)
}

func (s *Server) DiscoverSpaces(c echo.Context, params api.DiscoverSpacesParams) error {
	query := ""
	if params.Q != nil {
		query = *params.Q
	}
	spaces, err := s.store.DiscoverableSpaces(ctxOf(c), currentUser(c), query, maxDiscoverResults)
	if err != nil {
		return err
	}
	result := make([]api.DiscoverableSpaceDto, 0, len(spaces))
	for _, space := range spaces {
		dto := api.DiscoverableSpaceDto{Id: space.ID, Name: space.Name, Description: space.Description,
			IconId: space.IconImageID, MemberCount: space.MemberCount, JoinRole: api.JoinRoleEnum(space.JoinRole)}
		if space.Role != nil {
			role := api.SpaceMemberRoleEnum(*space.Role)
			dto.Role = &role
		}
		result = append(result, dto)
	}
	return c.JSON(http.StatusOK, result)
}

// JoinSpace lets anybody join a discoverable space. Other spaces answer 404, so their ids cannot be probed.
func (s *Server) JoinSpace(c echo.Context, spaceID api.ID) error {
	var role Role
	err := s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		space, err := tx.SpaceByID(ctxOf(c), spaceID)
		if err != nil {
			return orNotFound(err, "Space not found: %s", spaceID)
		}
		current, err := tx.MemberRole(ctxOf(c), spaceID, currentUser(c))
		if err == nil {
			role = Role(current)
			return nil
		}
		if !errors.Is(err, store.ErrNotFound) {
			return err
		}
		if !space.Discoverable {
			return notFound("Space not found: %s", spaceID)
		}
		role = Role(space.JoinRole)
		return joinSpace(ctxOf(c), tx, spaceID, currentUser(c), role)
	})
	if err != nil {
		return err
	}
	return s.spaceView(c, spaceID, role)
}

func (s *Server) DeleteSpace(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermDeleteSpace); err != nil {
		return err
	}
	err := s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		space, err := tx.SpaceByID(ctxOf(c), spaceID)
		if err != nil {
			return err
		}
		if err := tx.DeleteSpace(ctxOf(c), spaceID); err != nil {
			return err
		}
		return tx.DeleteImage(ctxOf(c), space.IconImageID)
	})
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) SetSpaceIcon(c echo.Context, spaceID api.ID) error {
	role, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermEditSpace)
	if err != nil {
		return err
	}
	contentType, data, err := s.readImage(c)
	if err != nil {
		return err
	}
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		space, err := tx.SpaceByID(ctxOf(c), spaceID)
		if err != nil {
			return err
		}
		imageID, err := tx.CreateImage(ctxOf(c), contentType, data)
		if err != nil {
			return err
		}
		if err := tx.SetSpaceIcon(ctxOf(c), spaceID, &imageID); err != nil {
			return err
		}
		return tx.DeleteImage(ctxOf(c), space.IconImageID)
	})
	if err != nil {
		return err
	}
	return s.spaceView(c, spaceID, role)
}

func (s *Server) RemoveSpaceIcon(c echo.Context, spaceID api.ID) error {
	role, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermEditSpace)
	if err != nil {
		return err
	}
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		space, err := tx.SpaceByID(ctxOf(c), spaceID)
		if err != nil {
			return err
		}
		if err := tx.SetSpaceIcon(ctxOf(c), spaceID, nil); err != nil {
			return err
		}
		return tx.DeleteImage(ctxOf(c), space.IconImageID)
	})
	if err != nil {
		return err
	}
	return s.spaceView(c, spaceID, role)
}

func (s *Server) LeaveSpace(c echo.Context, spaceID api.ID) error {
	err := s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		role, err := tx.MemberRole(ctxOf(c), spaceID, currentUser(c))
		if err != nil {
			return orNotFound(err, "Not a member of space: %s", spaceID)
		}
		if Role(role) == RoleOwner {
			if err := requireAnotherOwner(ctxOf(c), tx, spaceID, "The last owner cannot leave the space"); err != nil {
				return err
			}
		}
		return tx.RemoveMember(ctxOf(c), spaceID, currentUser(c))
	})
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func requireAnotherOwner(ctx context.Context, tx *store.Store, spaceID, message string) error {
	owners, err := tx.CountRole(ctx, spaceID, string(RoleOwner))
	if err != nil {
		return err
	}
	if owners <= 1 {
		return conflict("%s", message)
	}
	return nil
}

func (s *Server) GetSpaceMembers(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	members, err := s.store.Members(ctxOf(c), spaceID)
	if err != nil {
		return err
	}
	sort.SliceStable(members, func(i, j int) bool {
		ri, rj := Role(members[i].Role).rank(), Role(members[j].Role).rank()
		if ri != rj {
			return ri < rj
		}
		return members[i].Username < members[j].Username
	})
	result := make([]api.SpaceMemberDto, 0, len(members))
	for _, member := range members {
		result = append(result, memberDto(member))
	}
	return c.JSON(http.StatusOK, result)
}

func (s *Server) UpdateSpaceMember(c echo.Context, spaceID api.ID, userID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermManageMembers); err != nil {
		return err
	}
	var req api.UpdateSpaceMemberDto
	if err := bind(c, &req); err != nil {
		return err
	}
	newRole := Role(req.Role)
	if !validRole(newRole) {
		return badRequest("Unknown role: %s", req.Role)
	}
	var updated *store.Member
	err := s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		actor, target, err := s.managedMember(c, tx, spaceID, userID)
		if err != nil {
			return err
		}
		if !actor.AtLeast(newRole) {
			return forbidden("You cannot assign a role above your own")
		}
		if target == RoleOwner && newRole != RoleOwner {
			if err := requireAnotherOwner(ctxOf(c), tx, spaceID, "A space needs at least one owner"); err != nil {
				return err
			}
		}
		if err := tx.SetMemberRole(ctxOf(c), spaceID, userID, string(newRole)); err != nil {
			return err
		}
		updated, err = tx.Member(ctxOf(c), spaceID, userID)
		return err
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, memberDto(*updated))
}

func (s *Server) RemoveSpaceMember(c echo.Context, spaceID api.ID, userID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermManageMembers); err != nil {
		return err
	}
	err := s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		if _, _, err := s.managedMember(c, tx, spaceID, userID); err != nil {
			return err
		}
		return tx.RemoveMember(ctxOf(c), spaceID, userID)
	})
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// managedMember checks that the current user may manage the target: owners manage everybody else, everyone
// else only members ranked strictly below them. Changing your own membership goes through LeaveSpace.
func (s *Server) managedMember(c echo.Context, tx *store.Store, spaceID, userID string) (Role, Role, error) {
	actorRole, err := tx.MemberRole(ctxOf(c), spaceID, currentUser(c))
	if err != nil {
		return "", "", orNotFound(err, "Member not found: %s", currentUser(c))
	}
	targetRole, err := tx.MemberRole(ctxOf(c), spaceID, userID)
	if err != nil {
		return "", "", orNotFound(err, "Member not found: %s", userID)
	}
	actor, target := Role(actorRole), Role(targetRole)
	if userID == currentUser(c) {
		return "", "", badRequest("You cannot change your own membership; leave the space instead")
	}
	if !actor.Has(PermManageMembers) {
		return "", "", forbidden("Missing permission to manage members")
	}
	if actor != RoleOwner && actor.rank() >= target.rank() {
		return "", "", forbidden("You can only manage members with a lower role")
	}
	return actor, target, nil
}

// joinSpace adds the user to the space; joining never downgrades an existing membership.
func joinSpace(ctx context.Context, tx *store.Store, spaceID, userID string, role Role) error {
	current, err := tx.MemberRole(ctx, spaceID, userID)
	if errors.Is(err, store.ErrNotFound) {
		return tx.AddMember(ctx, spaceID, userID, string(role))
	}
	if err != nil {
		return err
	}
	if Role(current).AtLeast(role) {
		return nil
	}
	return tx.SetMemberRole(ctx, spaceID, userID, string(role))
}
