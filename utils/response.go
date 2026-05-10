package utils

import "github.com/gin-gonic/gin"

type response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Code    string `json:"code,omitempty"`
}

func Success(c *gin.Context, data any) {
	c.JSON(200, response{Success: true, Data: data})
}

func Error(c *gin.Context, httpCode int, msg string, code string) {
	c.JSON(httpCode, response{Success: false, Error: msg, Code: code})
}
