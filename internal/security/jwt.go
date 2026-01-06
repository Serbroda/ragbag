package security

import (
	"fmt"
	"time"

	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
	"github.com/Serbroda/ragbag/internal/utils"
	"github.com/golang-jwt/jwt/v5"
)

type Jwt = string

var (
	JwtSecretKey       = utils.MustGetEnv("JWT_SECRET_KEY")
	jwtAccessTokenExp  = 15
	jwtRefreshTokenExp = 10080
)

type JWTInfo struct {
	Token      *Jwt
	Expiration time.Time
}

type TokenPair struct {
	AccessToken  JWTInfo `json:"accessToken"`
	RefreshToken JWTInfo `json:"refreshToken"`
}

type JwtCustomClaims struct {
	Name  string `json:"name,omitempty"`
	Roles string `json:"roles,omitempty"`
}

func GenerateJwtPair(user sqlc.User) (TokenPair, error) {
	accessTokenExp := time.Now().Add(time.Minute * time.Duration(jwtAccessTokenExp))
	accessToken, err := GenerateJwt(jwt.MapClaims{
		"sub":   user.ID,
		"uid":   user.ID,
		"exp":   accessTokenExp.Unix(),
		"iat":   time.Now().Unix(),
		"roles": user.Role,
	})
	if err != nil {
		return TokenPair{}, err
	}

	refreshTokenExp := time.Now().Add(time.Minute * time.Duration(jwtRefreshTokenExp))
	refreshToken, err := GenerateJwt(jwt.MapClaims{
		"sub": user.ID,
		"uid": user.ID,
		"exp": refreshTokenExp.Unix(),
		"iat": time.Now().Unix(),
	})
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken: JWTInfo{
			Token:      &accessToken,
			Expiration: accessTokenExp,
		},
		RefreshToken: JWTInfo{
			Token:      &refreshToken,
			Expiration: refreshTokenExp,
		},
	}, nil
}

func GenerateJwt(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(JwtSecretKey))
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}

func ParseJwt(token string) (*jwt.Token, error) {
	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(JwtSecretKey), nil
	})
	return t, err
}
