package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logging gets the next request and records the time.
// Will only log with Info depth
func Logging() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()
		latency := time.Since(start)

		status := ctx.Writer.Status()
		slog.Info("request", "method", ctx.Request.Method, "path", ctx.Request.URL.Path, "status", status, "latency_ms", latency.Milliseconds())
	}
}
