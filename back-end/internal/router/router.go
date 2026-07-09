package router

import (
	"bookmyvenue/config"
	"bookmyvenue/internal/handler"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func SetupRouter(cfg *config.Config,rdb *redis.Client, authHandler *handler.AuthHandler,adminAuthHandler *handler.AdminAuthHandler,venueHandler *handler.VenueHandler) *gin.Engine {
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
		// S3 Presigned URL
		ownerRoutes.GET("/venues/presigned-url", venueHandler.GetPresignedURL)
		// Venue CRUD
		ownerRoutes.POST("/venues", venueHandler.CreateVenue)
		ownerRoutes.GET("/venues", venueHandler.GetOwnerVenues)
		ownerRoutes.GET("/venues/:id", venueHandler.GetVenueByID)
		ownerRoutes.PUT("/venues/:id", venueHandler.UpdateVenue)
		ownerRoutes.DELETE("/venues/:id", venueHandler.DeleteVenue)
		ownerRoutes.PATCH("/venues/:id/toggle", venueHandler.ToggleVenueStatus)
		// Space CRUD (nested under venue)
		ownerRoutes.POST("/venues/:id/spaces", venueHandler.AddSpace)
		ownerRoutes.PUT("/spaces/:id", venueHandler.UpdateSpace)
		ownerRoutes.DELETE("/spaces/:id", venueHandler.DeleteSpace)
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