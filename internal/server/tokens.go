package server

import (
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/auth"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// Personal API tokens let scripts and apps use the API as their user: "Authorization: Bearer rb_…". They are
// created in the settings (with the password), shown once and stored as a hash. A token reads (READ) or also
// changes (WRITE) links and collections of the user's spaces, and nothing else: the account, tokens, spaces'
// settings, members, invites, public links and who sees a collection stay with a signed-in person
// (tokenMay), so a token never gets around the password or two-factor authentication. Locked collections stay
// hidden for tokens. Blocked and deleted accounts lose theirs; changing the password does not revoke them.
// With plans (PLANS_ENABLED) tokens are part of Pro unless PLANS_FREE_API_TOKENS: Free accounts cannot create
// any, and tokens of an account that is Free again rest until it is Pro again (apiTokensFor).

const (
	apiTokenPrefix    = "rb_"
	scopeRead         = "READ"
	scopeWrite        = "WRITE"
	maxAPITokens      = 20
	maxTokenNameChars = 50
	maxTokenDays      = 3650
	// last_used_at is only written this often, not with every request
	tokenTouchInterval = time.Minute
)

// tokenNever: what no token may see or do, whatever its scope. Invites are links that let others join.
var tokenNever = []string{"spaces/discover", "spaces/*/invites"}

// tokenNoWrite: changes that manage a space or decide who sees what, left to signed-in people.
var tokenNoWrite = []string{
	"spaces", "spaces/*", "spaces/*/icon", "spaces/*/join", "spaces/*/leave", "spaces/*/members", "spaces/*/members/*",
	"spaces/*/trash/purge", "collections/*/access", "collections/*/share", "collections/*/lock",
}

// apiTokensFor tells whether the account may create and use API tokens.
func (s *Server) apiTokensFor(user *store.User) bool {
	return !s.cfg.Plans.Enabled || s.cfg.Plans.FreeAPITokens || s.proSource(user) != nil
}

const tokensNeedPro = "API tokens are part of Pro"

// tokenMay tells whether an API token of the scope may send this request. New endpoints are closed to tokens
// unless they are below spaces, collections or bookmarks.
func tokenMay(scope, method, requestPath string) bool {
	rest, ok := strings.CutPrefix(requestPath, "/api/v1/")
	if !ok {
		return false
	}
	read := method == http.MethodGet || method == http.MethodHead
	if rest == "users/me" {
		return read
	}
	if rest != "spaces" && !strings.HasPrefix(rest, "spaces/") && !strings.HasPrefix(rest, "collections/") &&
		!strings.HasPrefix(rest, "bookmarks/") {
		return false
	}
	if matchesAny(tokenNever, rest) {
		return false
	}
	if read {
		return true
	}
	return scope == scopeWrite && !matchesAny(tokenNoWrite, rest)
}

func matchesAny(patterns []string, value string) bool {
	for _, pattern := range patterns {
		if matched, _ := path.Match(pattern, value); matched {
			return true
		}
	}
	return false
}

// authenticateToken signs the request in as the owner of the API token, within what tokenMay allows.
func (s *Server) authenticateToken(c echo.Context, raw string) error {
	token, err := s.store.APITokenByHash(ctxOf(c), hashToken(raw))
	if err != nil {
		return unauthorized("Invalid or expired token")
	}
	if token.ExpiresAt != nil && time.Now().After(*token.ExpiresAt) {
		return unauthorized("Invalid or expired token")
	}
	user, err := s.store.UserByID(ctxOf(c), token.UserID)
	if err != nil || user.BlockedAt != nil || user.DeletedAt != nil || user.MustChangePassword {
		return unauthorized("Token has been revoked")
	}
	c.Set(userIDKey, user.ID)
	c.Set(usernameKey, user.Username)
	if !s.apiTokensFor(user) {
		return forbidden(tokensNeedPro + ", and this account is on the Free plan")
	}
	if !tokenMay(token.Scope, c.Request().Method, c.Request().URL.Path) {
		if token.Scope == scopeRead && tokenMay(scopeWrite, c.Request().Method, c.Request().URL.Path) {
			return forbidden("This API token only reads: create one that may also write")
		}
		return forbidden("API tokens work with links and collections; this needs somebody signed in")
	}
	if token.LastUsedAt == nil || time.Since(*token.LastUsedAt) > tokenTouchInterval {
		if err := s.store.TouchAPIToken(ctxOf(c), token.ID); err != nil {
			return err
		}
	}
	return nil
}

func apiTokenDto(token *store.APIToken) api.ApiTokenDto {
	return api.ApiTokenDto{Id: token.ID, Name: token.Name, Scope: api.ApiTokenScope(token.Scope),
		CreatedAt: token.CreatedAt, LastUsedAt: token.LastUsedAt, ExpiresAt: token.ExpiresAt}
}

func (s *Server) GetApiTokens(c echo.Context) error {
	tokens, err := s.store.APITokensOf(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	result := make([]api.ApiTokenDto, 0, len(tokens))
	for i := range tokens {
		result = append(result, apiTokenDto(&tokens[i]))
	}
	return c.JSON(http.StatusOK, result)
}

// CreateApiToken returns the token itself, once: afterwards only its hash is known.
func (s *Server) CreateApiToken(c echo.Context) error {
	var req api.CreateApiTokenRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := s.store.UserByID(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	// 400, not 401: the client would treat a 401 as an expired session
	if !auth.CheckPassword(user.PasswordHash, req.CurrentPassword) {
		return badRequest("The current password is wrong")
	}
	if !s.apiTokensFor(user) {
		return forbidden(tokensNeedPro + ": account menu → Plan")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > maxTokenNameChars {
		return badRequest("Token names must have 1 to %d characters", maxTokenNameChars)
	}
	if req.Scope != scopeRead && req.Scope != scopeWrite {
		return badRequest("The scope must be READ or WRITE")
	}
	token := &store.APIToken{UserID: user.ID, Name: name, Scope: string(req.Scope)}
	if req.ExpiresInDays != nil {
		if *req.ExpiresInDays < 1 || *req.ExpiresInDays > maxTokenDays {
			return badRequest("Tokens expire after 1 to %d days, or never", maxTokenDays)
		}
		token.ExpiresAt = ptr(time.Now().AddDate(0, 0, *req.ExpiresInDays))
	}
	existing, err := s.store.APITokensOf(ctxOf(c), user.ID)
	if err != nil {
		return err
	}
	if len(existing) >= maxAPITokens {
		return conflict("You can have at most %d API tokens: revoke one you no longer use", maxAPITokens)
	}
	raw := apiTokenPrefix + randomToken()
	token.TokenHash = hashToken(raw)
	if err := s.store.CreateAPIToken(ctxOf(c), token); err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "API token created", "token", token.Name, "scope", token.Scope)
	dto := apiTokenDto(token)
	return c.JSON(http.StatusOK, api.ApiTokenCreatedDto{Id: dto.Id, Name: dto.Name, Scope: dto.Scope,
		CreatedAt: dto.CreatedAt, ExpiresAt: dto.ExpiresAt, Token: raw})
}

func (s *Server) DeleteApiToken(c echo.Context, tokenID api.ID) error {
	found, err := s.store.DeleteAPIToken(ctxOf(c), currentUser(c), tokenID)
	if err != nil {
		return err
	}
	if !found {
		return notFound("API token not found: %s", tokenID)
	}
	audit(c, slog.LevelInfo, "API token revoked", "token", tokenID)
	return c.NoContent(http.StatusNoContent)
}
