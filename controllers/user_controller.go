package controllers

import (
	"errors"
	"net/http"

	"gin-hola-mundo/services"
	"gin-hola-mundo/utils"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	service services.UserService
}

func NewUserController(s services.UserService) *UserController {
	return &UserController{service: s}
}

func (uc *UserController) Create(c *gin.Context) {
	var req services.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, err.Error(), "VALIDATION_ERROR")
		return
	}

	user, err := uc.service.CreateUser(c.Request.Context(), req)
	if err != nil {
		var appErr *utils.AppError
		if errors.As(err, &appErr) {
			utils.Error(c, appErr.HTTPCode, appErr.Message, appErr.Code)
			return
		}
		utils.Error(c, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": user})
}
