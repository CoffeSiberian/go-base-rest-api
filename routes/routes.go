package routes

import (
	"gin-hola-mundo/config"
	"gin-hola-mundo/controllers"
	"gin-hola-mundo/middlewares"
	"gin-hola-mundo/repositories"
	"gin-hola-mundo/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	r.Use(middlewares.CORS())
	r.Use(gin.Recovery())

	userRepo := repositories.NewUserRepository(db)
	userSvc := services.NewUserService(userRepo)
	userCtrl := controllers.NewUserController(userSvc)

	api := r.Group("/api/v1")
	api.GET("/health", controllers.Health)

	users := api.Group("/users", middlewares.Auth(cfg.JWTSecret))
	users.POST("", userCtrl.Create)

	// Future route groups go here:
	// auth := api.Group("/auth")
}
