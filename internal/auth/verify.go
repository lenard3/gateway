package auth

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// VerifyAccessToken takes token+secret and tries to return claims
func VerifyAccessToken(tokenStr string, secret string) (*Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
		// throw out wrong signing methods. would use cpu time and is propably a crafted package
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth_VerifyAccessToken: unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	return &claims, nil
}
