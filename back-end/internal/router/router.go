package router

import (
	"bookmyvenue/config"
	"bookmyvenue/internal/handler"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func SetupRouter(cfg *config.Config,rdb *redis.Client, authHandler *handler.AuthHandler,adminAuthHandler *handler.AdminAuthHandler) *gin.Engine {
	r := gin.Default()

	loginLimiter := handler.RateLimiter(rdb, "login", 5, 15*time.Minute)
	registerLimiter := handler.RateLimiter(rdb, "register", 3, 1*time.Hour)

    // Public Auth Routes (User/Owner)
	auth := r.Group("/api/auth")
	{
		auth.POST("/register/user", registerLimiter, authHandler.RegisterUser)
		auth.POST("/register/owner", registerLimiter, authHandler.RegisterOwner)
		auth.POST("/login", loginLimiter, authHandler.Login)
	}

	// Public Auth Routes (Admin)
	adminAuth := r.Group("/api/admin/auth")
	{
		adminAuth.POST("/login", loginLimiter, adminAuthHandler.Login)
	}

	// Protected User/Owner Routes
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

	// Protected Owner Routes
	ownerRoutes := r.Group("/api/owner")
	ownerRoutes.Use(handler.AuthMiddleware(cfg))
	ownerRoutes.Use(handler.RoleMiddleware("owner"))
	{
		ownerRoutes.GET("/dashboard", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "owner dashboard — only owners can see this"})
		})
	}

	// Protected Admin Routes
	adminRoutes := r.Group("/api/admin")
	adminRoutes.Use(handler.AuthMiddleware(cfg))
	adminRoutes.Use(handler.RoleMiddleware("admin"))
	{
		adminRoutes.GET("/dashboard", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "admin dashboard — only admins can see this"})
		})
	}

	return r
}