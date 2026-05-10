package middlewares

import (
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		gin.DefaultWriter.Write([]byte(
			time.Now().Format(time.RFC3339) + " | " +
				c.Request.Method + " " + c.Request.URL.Path +
				" | " + c.ClientIP() +
				" | status=" + string(rune(c.Writer.Status())) +
				" | latency=" + time.Since(start).String() + "\n",
		))
	}
}
