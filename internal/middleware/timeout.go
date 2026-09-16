package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// Timeout sets a global per request timeout.
func Timeout(d time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(ctx.Request.Context(), d)
		defer cancel()
		ctx.Request = ctx.Request.WithContext(c)
		ctx.Next()
	}
}
