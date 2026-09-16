package httperr

import "github.com/gin-gonic/gin"

func Respond(c *gin.Context, status int, code string, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
