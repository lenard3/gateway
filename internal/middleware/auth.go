package middleware

import (
	"gateway/internal/auth"
	"gateway/internal/httperr"
	"gateway/internal/store"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const tokenType = "Bearer"

// Auth returns a func that checks each incoming request header.
// Auth header gets checked for token type and token.
// Token is then verified.
// Basically just a gatekeeping func to sort out obvious errors in requests
func Auth(secret string, tokenStore *store.TokenStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		// we don't allow empty auth headers since every request needs a token anyway
		if authHeader == "" {
			httperr.Respond(ctx, http.StatusUnauthorized, "missing_token", "authorization header required")
			return
		}

		// check token type and error out if necessary
		parts := strings.SplitN(authHeader, " ", 2)
		if ok := strings.EqualFold(parts[0], tokenType); !ok {
			httperr.Respond(ctx, http.StatusUnauthorized, "invalid_auth_header", "auth header invalid")
			return
		}

		if len(parts) != 2 || parts[1] == "" {
			httperr.Respond(ctx, http.StatusUnauthorized, "invalid_auth_header", "auth header invalid")
			return
		}

		claims, err := auth.VerifyAccessToken(parts[1], secret)
		if err != nil {
			httperr.Respond(ctx, http.StatusUnauthorized, "invalid_token", "token verification failed")
			slog.Debug("token verification failed", "error", err)
			return
		}
		blocked, err := tokenStore.IsBlocked(ctx.Request.Context(), claims.ID)
		if err != nil {
			httperr.Respond(ctx, http.StatusInternalServerError, "internal_error", "token validity can't be checked")
			slog.Debug("blocklist can't be checked", "error", err)
			return
		}
		if blocked {
			httperr.Respond(ctx, http.StatusUnauthorized, "token_revoked", "token has been revoked")
			return
		}

		ctx.Set("userID", claims.Subject)
		ctx.Next()
	}
}
