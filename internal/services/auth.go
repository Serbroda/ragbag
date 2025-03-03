package services

import (
	"context"
	"database/sql"
	"errors"
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
	"github.com/Serbroda/ragbag/internal/security"
	"github.com/golang-jwt/jwt/v5"
	"github.com/teris-io/shortid"
	"strings"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrWrongPassword     = errors.New("wrong password")
	ErrInvalidToken      = errors.New("invalid token")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type AuthService interface {
	Register(ctx context.Context, params sqlc.InsertUserParams) (sqlc.User, error)
	Login(ctx context.Context, email, password string) (security.TokenPair, error)
	RefreshToken(ctx context.Context, token string) (security.TokenPair, error)
}

type authService struct {
	queries      *sqlc.Queries
	spaceService SpaceService
}

func NewAuthService(queries *sqlc.Queries, spaceService SpaceService) AuthService {
	return &authService{queries: queries, spaceService: spaceService}
}

func (a authService) Register(ctx context.Context, params sqlc.InsertUserParams) (sqlc.User, error) {
	_, err := a.queries.FindUserByEmail(ctx, params.Email)
	if err == nil {
		return sqlc.User{}, ErrUserAlreadyExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return sqlc.User{}, err
	}

	params.Sid = shortid.MustGenerate()
	params.Email = strings.TrimSpace(strings.ToLower(params.Email))

	if !security.IsBcryptHash(params.Password) {
		encryptedPassword, err := security.HashBcrypt(params.Password)
		if err != nil {
			return sqlc.User{}, err
		}
		params.Password = encryptedPassword
	}

	user, err := a.queries.InsertUser(ctx, params)
	if err != nil {
		return sqlc.User{}, err
	}

	_, err = a.spaceService.Create(ctx, sqlc.InsertSpaceParams{
		OwnerID: user.ID,
		Name:    "Default",
	})
	if err != nil {
		return sqlc.User{}, err
	}
	return user, nil
}

func (a authService) Login(ctx context.Context, email, password string) (security.TokenPair, error) {
	entity, err := a.queries.FindUserByEmail(ctx, email)
	if err != nil {
		return security.TokenPair{}, ErrUserNotFound
	}

	if !security.CheckBcryptHash(password, entity.Password) {
		return security.TokenPair{}, ErrWrongPassword
	}

	return security.GenerateJwtPair(entity)
}

func (a authService) RefreshToken(ctx context.Context, token string) (security.TokenPair, error) {
	t, err := security.ParseJwt(token)
	if err != nil {
		return security.TokenPair{}, err
	}

	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok || !t.Valid {
		return security.TokenPair{}, ErrInvalidToken
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return security.TokenPair{}, ErrInvalidToken
	}

	entity, err := a.queries.FindUserBySid(ctx, sub)
	if err != nil {
		return security.TokenPair{}, ErrUserNotFound
	}

	return security.GenerateJwtPair(entity)
}
