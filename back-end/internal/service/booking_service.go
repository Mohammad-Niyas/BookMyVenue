package service

import (
	"bookmyvenue/internal/domain"
	"bookmyvenue/internal/repository"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type BookingRequest struct {
	SpaceID uuid.UUID `json:"space_id" binding:"required"`
	SlotID  uuid.UUID `json:"slot_id" binding:"required"`
}

type BookingResponse struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	SpaceID     uuid.UUID `json:"space_id"`
	SlotID      uuid.UUID `json:"slot_id"`
	TotalAmount float64   `json:"total_amount"`
	Status      string    `json:"status"`
}

type BookingService interface{
	CreateBooking(ctx context.Context,userID uuid.UUID, req BookingRequest) (*BookingResponse, error)
}

type bookingService struct{
	bookingRepo repository.BookingRepository
	spaceRepo repository.SpaceRepository
	venueRepo repository.VenueRepository
	rdb *redis.Client
}

func NewBookingService(bookingRepo repository.BookingRepository,spaceRepo repository.SpaceRepository,venueRepo repository.VenueRepository,rdb *redis.Client) BookingService{
	return &bookingService{
		bookingRepo: bookingRepo,
		spaceRepo: spaceRepo,
		venueRepo: venueRepo,
		rdb: rdb,
	}
}

func (s *bookingService) CreateBooking(ctx context.Context,userID uuid.UUID, req BookingRequest) (*BookingResponse, error){
	space,err:=s.spaceRepo.FindBySpaceID(req.SpaceID)
	if err!=nil{
		return nil, errors.New("Space not found")
	}
	venue,err:=s.venueRepo.FindByID(space.VenueID)
	if err!=nil{
		return nil,errors.New("Venue Not Found")
	}
	if venue.Status != "approved"{
		return nil,errors.New("Venues not active")
	}
	slot,err:=s.spaceRepo.FindBySlotID(req.SlotID)
	if err!=nil{
		return nil,errors.New("slot not found")
	}
	if slot.IsBooked{
		return nil,errors.New("this slot already booked")
	}
	actualPrice := space.Price
	if slot.Price != nil {
    	actualPrice = *slot.Price
	}
	if slot.Date.Before(time.Now().Truncate(24 * time.Hour)) {
    	return nil, errors.New("cannot book a slot in the past")
	}
	if slot.SpaceID != req.SpaceID {
    	return nil, errors.New("slot does not belong to this space")
	}

	redisKey := "hold:slot:" + req.SlotID.String()
	locked, err := s.rdb.SetNX(ctx, redisKey, userID.String(), 10*time.Minute).Result()
	if err != nil {
    	return nil, errors.New("failed to acquire booking hold due to server error")
	}
	if !locked {
    	return nil, errors.New("this slot is currently being held by another user")
	}

	booking:=domain.Booking{
		UserID : userID,
		SpaceID: req.SpaceID,
		SlotID: req.SlotID,
		TotalAmount: actualPrice,
		AmountPaid: 0.0,
		Status: "pending",
	}

	if err:=s.bookingRepo.Create(ctx,&booking);err!=nil{
		s.rdb.Del(ctx, redisKey)
		return nil,errors.New("booking created filed")
	}
	response := mapToBookingResponse(booking)
	return &response, nil
}

func mapToBookingResponse(b domain.Booking) BookingResponse {
    return BookingResponse{
        ID:          b.ID,
        UserID:      b.UserID,
        SpaceID:     b.SpaceID,
        SlotID:      b.SlotID,
        TotalAmount: b.TotalAmount,
        Status:      b.Status,
    }
}