package handler

import (
	"bookmyvenue/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VenueHandler struct {
	venueService service.VenueService
}

func NewVenueHandler(venueService service.VenueService) *VenueHandler {
	return &VenueHandler{venueService: venueService}
}

func getOwnerID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("user ID not found in token")
	}
	return userID.(uuid.UUID), nil
}

// Venue Endpoints

func (h *VenueHandler) CreateVenue(c *gin.Context) {
	ownerID, err := getOwnerID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}
	var req service.CreateVenueRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	venue, err := h.venueService.CreateVenue(ownerID, req)
	if err != nil {
		if err.Error() == "a venue with this name and address already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "venue must have between 4 and 10 images" || 
		   err.Error() == "invalid timings format: must be HH:MM-HH:MM (e.g., 08:00-22:00)" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "venue submitted for approval",
		"data":    venue,
	})
}

func (h *VenueHandler) GetOwnerVenues(c *gin.Context) {
	ownerID, err := getOwnerID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}
	venues, err := h.venueService.GetOwnerVenues(ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "venues fetched successfully",
		"count":   len(venues),
		"data":    venues,
	})
}

func (h *VenueHandler) GetVenueByID(c *gin.Context) {
	ownerID, err := getOwnerID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}
	venueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid venue ID"})
		return
	}
	venue, err := h.venueService.GetVenueByID(ownerID, venueID)
	if err != nil {
		if err.Error() == "venue not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "unauthorized: you don't own this venue" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "venue fetched successfully",
		"data":    venue,
	})
}

func (h *VenueHandler) UpdateVenue(c *gin.Context) {
	ownerID, err := getOwnerID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}

	venueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid venue ID"})
		return
	}

	var req service.UpdateVenueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	venue, isDraft, err := h.venueService.UpdateVenue(ownerID, venueID, req)
	if err != nil {
		if err.Error() == "venue not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "unauthorized: you don't own this venue" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if isDraft {
		c.JSON(http.StatusOK, gin.H{
			"message": "changes submitted for admin approval",
			"data":    venue,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "venue updated successfully",
		"data":    venue,
	})
}

func (h *VenueHandler) DeleteVenue(c *gin.Context) {
	ownerID, err := getOwnerID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}
	venueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid venue ID"})
		return
	}
	err = h.venueService.DeleteVenue(ownerID, venueID)
	if err != nil {
		if err.Error() == "venue not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "unauthorized: you don't own this venue" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "venue deleted successfully",
	})
}

// Space Endpoints

func (h *VenueHandler) AddSpace(c *gin.Context) {
	ownerID, err := getOwnerID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}
	venueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid venue ID"})
		return
	}
	var req service.CreateSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	space, err := h.venueService.AddSpace(ownerID, venueID, req)
	if err != nil {
		if err.Error() == "venue not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "unauthorized: you don't own this venue" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "space added successfully",
		"data":    space,
	})
}

func (h *VenueHandler) UpdateSpace(c *gin.Context) {
	ownerID, err := getOwnerID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}
	spaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid space ID"})
		return
	}
	var req service.CreateSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	space, err := h.venueService.UpdateSpace(ownerID, spaceID, req)
	if err != nil {
		if err.Error() == "space not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "unauthorized: you don't own this venue" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "space updated successfully",
		"data":    space,
	})
}

func (h *VenueHandler) DeleteSpace(c *gin.Context) {
	ownerID, err := getOwnerID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}
	spaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid space ID"})
		return
	}
	err = h.venueService.DeleteSpace(ownerID, spaceID)
	if err != nil {
		if err.Error() == "space not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "unauthorized: you don't own this venue" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "space deleted successfully",
	})
}

// S3

func (h *VenueHandler) GetPresignedURL(c *gin.Context) {
	fileName := c.Query("filename")
	contentType := c.Query("content_type")
	if fileName == "" || contentType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename and content_type query params are required"})
		return
	}
	if contentType != "image/jpeg" && contentType != "image/png" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only image/jpeg and image/png are allowed"})
		return
	}
	result, err := h.venueService.GeneratePresignedURL(c.Request.Context(), fileName, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "presigned URL generated successfully",
		"data":    result,
	})
}


