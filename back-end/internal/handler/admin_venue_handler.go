package handler

import (
	"bookmyvenue/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminVenueHandler struct {
	adminService service.AdminVenueService
}

func NewAdminVenueHandler(adminService service.AdminVenueService) *AdminVenueHandler {
	return &AdminVenueHandler{adminService: adminService}
}

func (h *AdminVenueHandler) GetPendingVenues(c *gin.Context) {
	venues, err := h.adminService.GetPendingVenues()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "pending venues fetched successfully",
		"count":   len(venues),
		"data":    venues,
	})
}

func (h *AdminVenueHandler) ApproveVenue(c *gin.Context) {
	venueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid venue ID"})
		return
	}
	err = h.adminService.ApproveVenue(venueID)
	if err != nil {
		if err.Error() == "venue not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "venue approved successfully: it is now active",
	})
}

func (h *AdminVenueHandler) RejectVenue(c *gin.Context) {
	venueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid venue ID"})
		return
	}
	var req service.RejectVenueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = h.adminService.RejectVenue(venueID, req.Reason)
	if err != nil {
		if err.Error() == "venue not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "venue rejected successfully",
	})
}

func (h *AdminVenueHandler) GetPendingDrafts(c *gin.Context) {
	drafts, err := h.adminService.GetPendingDrafts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "pending edit drafts fetched successfully",
		"count":   len(drafts),
		"data":    drafts,
	})
}

func (h *AdminVenueHandler) ApproveEditDraft(c *gin.Context) {
	draftID, err := uuid.Parse(c.Param("draft_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid draft ID"})
		return
	}
	err = h.adminService.ApproveEditDraft(draftID)
	if err != nil {
		if err.Error() == "edit draft not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "venue draft approved successfully: changes merged to live table",
	})
}

func (h *AdminVenueHandler) RejectEditDraft(c *gin.Context) {
	draftID, err := uuid.Parse(c.Param("draft_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid draft ID"})
		return
	}
	var req service.RejectDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = h.adminService.RejectEditDraft(draftID, req.AdminNote)
	if err != nil {
		if err.Error() == "edit draft not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "venue draft rejected successfully",
	})
}
