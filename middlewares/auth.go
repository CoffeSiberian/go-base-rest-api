package middlewares

import (
	"strings"

	"gin-hola-mundo/utils"

	"github.com/gin-gonic/gin"
)

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			utils.Error(c, 401, "Authorization header required", "UNAUTHORIZED")
			c.Abort()
			return
		}

		claims, err := utils.ValidateClaims(strings.TrimPrefix(header, "Bearer "), secret)
		if err != nil {
			utils.Error(c, 401, "Invalid or expired token", "UNAUTHORIZED")
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
