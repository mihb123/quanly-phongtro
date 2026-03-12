package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/mihb123/quanly-phongtro/config"
	"github.com/mihb123/quanly-phongtro/src/Controller"
	"github.com/mihb123/quanly-phongtro/src/Middleware"
	"github.com/mihb123/quanly-phongtro/src/Repository"
	"github.com/mihb123/quanly-phongtro/src/Service"
)

func Register(router *gin.Engine, configuration *config.Config, databaseConnection *sql.DB) {
	userRepository := repository.NewUserRepository(databaseConnection)
	authService := service.NewAuthService(userRepository)
	authController := controller.NewAuthController(authService)

	router.GET("/", controller.Home)

	apiGroup := router.Group("/api/v1")
	apiGroup.Use(middleware.JSONContentType())
	{
		apiGroup.GET("/health", controller.HealthCheck)

		authGroup := apiGroup.Group("/auth")
		{
			authGroup.POST("/register", authController.Register)
			authGroup.POST("/login", authController.Login)
		}
	}
}
