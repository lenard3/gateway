package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	jwt.RegisteredClaims
}

func IssueAccessToken(userID string, secret string, ttl time.Duration) (string, error) {
	jti := uuid.NewString()
	claims := jwt.RegisteredClaims{Subject: userID, IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)), ID: jti, Issuer: "gateway"}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("auth_IssueAccessToken: failed to sign token")
	}

	return signed, nil
}

func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("auth_GenerateRefreshToken: failed to generate refresh token: %w", err)
	}
	return base64.RawStdEncoding.EncodeToString(b), nil
}
