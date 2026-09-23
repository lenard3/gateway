package auth

import (
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
