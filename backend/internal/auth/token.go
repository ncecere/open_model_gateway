package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenPair represents access and refresh tokens with expiry metadata.
type TokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
	RefreshTokenID   string
}

type TokenManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	issuer     string
}

func NewTokenManager(secret string, accessTTL, refreshTTL time.Duration, issuer string) (*TokenManager, error) {
	if secret == "" {
		return nil, errors.New("token secret required")
	}
	if accessTTL <= 0 {
		return nil, errors.New("access ttl must be > 0")
	}
	if refreshTTL <= 0 {
		return nil, errors.New("refresh ttl must be > 0")
	}
	return &TokenManager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		issuer:     issuer,
	}, nil
}

func (tm *TokenManager) Generate(userID uuid.UUID, email string) (*TokenPair, error) {
	now := time.Now()
	accessExp := now.Add(tm.accessTTL)
	refreshExp := now.Add(tm.refreshTTL)

	accessClaims := jwt.MapClaims{
		"sub":   userID.String(),
		"email": email,
		"iat":   now.Unix(),
		"exp":   accessExp.Unix(),
		"iss":   tm.issuer,
		"typ":   "access",
		"jti":   uuid.NewString(),
	}
	accessToken, err := tm.sign(accessClaims)
	if err != nil {
		return nil, err
	}

	refreshID := uuid.NewString()
	refreshClaims := jwt.MapClaims{
		"sub": userID.String(),
		"iat": now.Unix(),
		"iss": tm.issuer,
		"exp": refreshExp.Unix(),
		"typ": "refresh",
		"jti": refreshID,
	}
	refreshToken, err := tm.sign(refreshClaims)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExp,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExp,
		RefreshTokenID:   refreshID,
	}, nil
}

func (tm *TokenManager) sign(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(tm.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func GenerateState(size int) (string, error) {
	if size <= 0 {
		size = 32
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
