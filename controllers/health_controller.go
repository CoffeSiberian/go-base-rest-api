package controllers

import (
	"gin-hola-mundo/utils"

	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) {
	utils.Success(c, gin.H{"status": "ok"})
}
