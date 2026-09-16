package httperr

import "github.com/gin-gonic/gin"

// Respond returns a formatted json error message
// Format:
//
//	{
//	  "error": {
//	    "code": code,
//	    "message": message,
//	  }
//	}
func Respond(c *gin.Context, status int, code string, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
