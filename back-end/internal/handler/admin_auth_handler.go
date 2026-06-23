package handler

import (
	"bookmyvenue/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminAuthHandler struct {
	adminAuthService service.AdminAuthService
}

func NewAdminAuthHandler(adminAuthService service.AdminAuthService) *AdminAuthHandler {
	return &AdminAuthHandler{adminAuthService: adminAuthService}
}

func (h *AdminAuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
		return
	}

	response, err := h.adminAuthService.Login(req)

	if err != nil {
		if err.Error() == "invalid email or password" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "admin login successful",
		"data":    response,
	})
}