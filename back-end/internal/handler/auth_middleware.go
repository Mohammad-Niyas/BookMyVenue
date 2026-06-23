package handler

import (
	"bookmyvenue/config"
	"bookmyvenue/pkg/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header is required"})
			c.Abort()
			return
		}
		
		parts := strings.Split(authHeader, " ")

		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format. Use: Bearer <token>"})
			c.Abort()
			return
		}
		
		tokenStr := parts[1]

		claims, err := utils.ValidateToken(tokenStr, cfg.JWTSecret)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		roleStr := userRole.(string)
		for _, role := range allowedRoles {
			if roleStr == role {
				c.Next()
				return
			}
		}
		
		c.JSON(http.StatusForbidden, gin.H{"error": "you don't have permission to access this resource"})
		c.Abort()
	}
}