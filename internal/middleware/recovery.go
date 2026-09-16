package middleware

import (
	"fmt"
	"gateway/internal/httperr"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Recovery catches panics so that the gateway doesnt crash.
// Returns an internal server error instead
func Recovery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// When handler returns, does nothing or catches panic and returns http error
		defer func() {
			// recover catches panic and returns panic, so that it can be handled (basically `except:`)
			if r := recover(); r != nil {
				slog.Error("panic recovered", "error", fmt.Sprint(r), "path", ctx.Request.URL.Path)
				httperr.Respond(ctx, http.StatusInternalServerError, "internal_error", "internal server error")
			}
		}()
		ctx.Next()
	}
}
