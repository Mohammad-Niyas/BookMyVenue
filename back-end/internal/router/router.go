package router

import (
	"bookmyvenue/config"
	"bookmyvenue/internal/handler"

	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config, authHandler *handler.AuthHandler) *gin.Engine {
	r := gin.Default()

	auth := r.Group("/api/auth")
	{
		auth.POST("/register/user", authHandler.RegisterUser)
		auth.POST("/register/owner", authHandler.RegisterOwner)
		auth.POST("/login", authHandler.Login)
	}

	userRoutes := r.Group("/api/user")
	userRoutes.Use(handler.AuthMiddleware(cfg))
	userRoutes.Use(handler.RoleMiddleware("user", "owner"))
	{
		userRoutes.GET("/profile", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			role, _ := c.Get("user_role")
			c.JSON(200, gin.H{
				"message": "protected route working!",
				"user_id": userID,
				"role":    role,
			})
		})
	}

	ownerRoutes := r.Group("/api/owner")
	ownerRoutes.Use(handler.AuthMiddleware(cfg))
	ownerRoutes.Use(handler.RoleMiddleware("owner"))
	{
		ownerRoutes.GET("/dashboard", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "owner dashboard — only owners can see this"})
		})
	}

	return r
}