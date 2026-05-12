package service

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestTokenIssuerGenerateToken(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.Secret = "12345678901234567890123456789012"
	cfg.JWT.AccessTokenExpireMinutes = 15

	issuer := NewTokenIssuer(cfg)
	user := &User{
		ID:                   42,
		Email:                "admin@example.com",
		Role:                 RoleAdmin,
		TokenVersion:         7,
		TokenVersionResolved: true,
	}

	tokenString, err := issuer.GenerateToken(user)
	require.NoError(t, err)

	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWT.Secret), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)
	require.Equal(t, user.ID, claims.UserID)
	require.Equal(t, user.Email, claims.Email)
	require.Equal(t, user.Role, claims.Role)
	require.Equal(t, user.TokenVersion, claims.TokenVersion)
	require.WithinDuration(t, time.Now().Add(15*time.Minute), claims.ExpiresAt.Time, 2*time.Second)
}

func TestTokenIssuerGetAccessTokenExpiresInFallsBackToHours(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.Secret = "12345678901234567890123456789012"
	cfg.JWT.ExpireHour = 2

	issuer := NewTokenIssuer(cfg)

	require.Equal(t, 7200, issuer.GetAccessTokenExpiresIn())
}
