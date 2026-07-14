package handler

import (
	"bookmyvenue/internal/service"
	"errors"
	"fmt"
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

func (h *VenueHandler) ToggleVenueStatus(c *gin.Context) {
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
	venue, err := h.venueService.ToggleVenueStatus(ownerID, venueID)
	if err != nil {
		if err.Error() == "venue not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "unauthorized: you don't own this venue" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "venue status toggled successfully",
		"data":    venue,
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

// Venue Search

func (h *VenueHandler) SearchVenues(c *gin.Context) {
	limitVal := c.DefaultQuery("limit", "10")
	offsetVal := c.DefaultQuery("offset", "0")

	city := c.Query("city")
	venueType := c.Query("type")
	query := c.Query("query")
	bookingType := c.Query("booking_type")

	var minPrice, maxPrice float64
	var minCapacity int
	
	if val := c.Query("min_price"); val != "" {
		fmt.Sscanf(val, "%f", &minPrice)
	}
	if val := c.Query("max_price"); val != "" {
		fmt.Sscanf(val, "%f", &maxPrice)
	}
	if val := c.Query("min_capacity"); val != "" {
		fmt.Sscanf(val, "%d", &minCapacity)
	}

	var limit, offset int
	fmt.Sscanf(limitVal, "%d", &limit)
	fmt.Sscanf(offsetVal, "%d", &offset)

	if limit <= 0 || limit > 50 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	venues, count, err := h.venueService.SearchVenues(city, venueType, query, minPrice, maxPrice, minCapacity,bookingType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "venues searched successfully",
		"total":   count,
		"limit":   limit,
		"offset":  offset,
		"data":    venues,
	})
}

func (h *VenueHandler) GetPublicVenueByID(c *gin.Context) {
	venueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid venue ID"})
		return
	}

	venue, err := h.venueService.GetPublicVenueByID(venueID)
	if err != nil {
		if err.Error() == "venue not found" || err.Error() == "venue is not available" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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

func (h *VenueHandler) GenerateSlots(c *gin.Context) {
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
	var req service.GenerateSlotsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	slots, err := h.venueService.GenerateSlots(ownerID, spaceID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "slots generated successfully",
		"count":   len(slots),
		"data":    slots,
	})
}

func (h *VenueHandler) GetAvailableSlots(c *gin.Context) {
	spaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid space ID"})
		return
	}
	dateStr := c.Query("date")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date query parameter is required (YYYY-MM-DD)"})
		return
	}
	slots, err := h.venueService.GetAvailableSlots(spaceID, dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "available slots fetched successfully",
		"count":   len(slots),
		"data":    slots,
	})
}



