package handler

import (
	"bookmyvenue/internal/service"
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BookingHandler struct{
	BookingService service.BookingService
}

func NewBookingHandler(BookingService service.BookingService)*BookingHandler{
	return &BookingHandler{
		BookingService: BookingService,
	}
}

func getUserID(c *gin.Context)(uuid.UUID, error){
	userId,exist:=c.Get("user_id")
	if !exist{
		return uuid.Nil,errors.New("user not found in token")
	}
	return userId.(uuid.UUID),nil
}

func (h *BookingHandler) CreateBooking(c *gin.Context){
	userID,err:=getUserID(c)
	if err!=nil{
		c.JSON(http.StatusUnauthorized,gin.H{"error":"Invalid user ID"})
		return
	}

	var req service.BookingRequest

	if err:=c.ShouldBindJSON(&req);err!=nil{
		c.JSON(http.StatusBadRequest,gin.H{"error":err.Error()})
		return
	}

	ctx,cancel:=context.WithTimeout(c.Request.Context(),5*time.Second)
	defer cancel()

	booking,err:=h.BookingService.CreateBooking(ctx,userID,req)

	if err!=nil{
		if err.Error()=="this slot already booked"{
			c.JSON(http.StatusConflict,gin.H{"error":err.Error()})
			return
		}
		if err.Error()=="Space not found"{
			c.JSON(http.StatusNotFound,gin.H{"error":err.Error()})
			return
		}
		if err.Error()=="Venue Not Found"{
			c.JSON(http.StatusNotFound,gin.H{"error":err.Error()})
			return
		}
		if err.Error()=="slot not found"{
			c.JSON(http.StatusNotFound,gin.H{"error":err.Error()})
			return
		}
		if err.Error()=="slot does not belong to this space"{
			c.JSON(http.StatusBadRequest,gin.H{"error":err.Error()})
			return
		}
		if err.Error() == "this slot is currently being held by another user" {
    		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
    		return
		}
		if err.Error()=="failed to acquire booking hold due to server error"{
			c.JSON(http.StatusInternalServerError,gin.H{"error":err.Error()})
			return 
		}
		if err.Error()=="daily venues (auditoriums/banquet halls) must be booked at least 30 days in advance"{
			c.JSON(http.StatusBadRequest,gin.H{"error":err.Error()})
			return 
		}
		if err.Error()=="booking created filed"{
			c.JSON(http.StatusInternalServerError,gin.H{"error":err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    	return
	}
	c.JSON(http.StatusCreated,gin.H{
		"message":"Booking hold acquired. Please complete payment within 10 minutes.",
		"data":booking,
	})
}

func (h *BookingHandler) CancelBooking(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID from token"})
		return
	}

	bookingIDStr := c.Param("id")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID format: must be a valid UUID"})
		return
	}

	var req service.CancelBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	res, err := h.BookingService.CancelBooking(ctx, userID, bookingID, req)
	if err != nil {
		if err.Error() == "booking not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "unauthorized: you do not own this booking" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "booking is already cancelled" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Booking cancelled successfully",
		"data":    res,
	})
}
