package service

import (
	"bookmyvenue/internal/domain"
	"bookmyvenue/internal/repository"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// releaseLockScript ensures atomic release of a distributed lock:
// It checks whether the key's value matches the given userID before deleting.
var releaseLockScript = redis.NewScript(`
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	else
		return 0
	end
`)

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

type CancelBookingRequest struct {
	Reason string `json:"reason" binding:"required,min=5,max=255"`
}

type CancelBookingResponse struct {
	BookingID          uuid.UUID `json:"booking_id"`
	Status             string    `json:"status"`
	RefundAmount       float64   `json:"refund_amount"`
	RefundPercentage   float64   `json:"refund_percentage"`
	CancellationReason string    `json:"cancellation_reason"`
	CancelledAt        time.Time `json:"cancelled_at"`
}

type BookingService interface {
	CreateBooking(ctx context.Context, userID uuid.UUID, req BookingRequest) (*BookingResponse, error)
	CancelBooking(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID, req CancelBookingRequest) (*CancelBookingResponse, error)
}

type bookingService struct {
	bookingRepo repository.BookingRepository
	spaceRepo   repository.SpaceRepository
	venueRepo   repository.VenueRepository
	rdb         *redis.Client
	db          *gorm.DB
}

func NewBookingService(bookingRepo repository.BookingRepository, spaceRepo repository.SpaceRepository, venueRepo repository.VenueRepository, rdb *redis.Client, db *gorm.DB) BookingService {
	return &bookingService{
		bookingRepo: bookingRepo,
		spaceRepo:   spaceRepo,
		venueRepo:   venueRepo,
		rdb:         rdb,
		db:          db,
	}
}

func (s *bookingService) CreateBooking(ctx context.Context, userID uuid.UUID, req BookingRequest) (*BookingResponse, error) {
	space, err := s.spaceRepo.FindBySpaceID(req.SpaceID)
	if err != nil {
		return nil, errors.New("Space not found")
	}
	venue, err := s.venueRepo.FindByID(space.VenueID)
	if err != nil {
		return nil, errors.New("Venue Not Found")
	}
	if venue.Status != "approved" {
		return nil, errors.New("Venues not active")
	}
	slot, err := s.spaceRepo.FindBySlotID(req.SlotID)
	if err != nil {
		return nil, errors.New("slot not found")
	}
	if slot.IsBooked {
		return nil, errors.New("this slot already booked")
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

	if space.BookingType == "daily" && actualPrice >= 10000.00 {
		minBookDate := time.Now().AddDate(0, 0, 30).Truncate(24 * time.Hour)
		if slot.Date.Before(minBookDate) {
			return nil, errors.New("daily venues over ₹10,000 must be booked at least 30 days in advance")
		}
	}

	redisKey := "hold:slot:" + req.SlotID.String()
	locked, err := s.rdb.SetNX(ctx, redisKey, userID.String(), 10*time.Minute).Result()
	if err != nil {
		return nil, errors.New("failed to acquire booking hold due to server error")
	}
	if !locked {
		return nil, errors.New("this slot is currently being held by another user")
	}

	booking := domain.Booking{
		UserID:      userID,
		SpaceID:     req.SpaceID,
		SlotID:      req.SlotID,
		TotalAmount: actualPrice,
		AmountPaid:  0.0,
		Status:      "pending",
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.bookingRepo.Create(ctx, tx, &booking); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		// Atomic safe release: only delete if the lock still belongs to this user
		_ = releaseLockScript.Run(ctx, s.rdb, []string{redisKey}, userID.String()).Err()
		return nil, errors.New("failed to create booking record")
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

func (s *bookingService) CancelBooking(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID, req CancelBookingRequest) (*CancelBookingResponse, error) {
	// 1. Fetch Booking Record
	booking, err := s.bookingRepo.FindByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("booking not found")
		}
		return nil, errors.New("failed to fetch booking details")
	}

	// 2. IDOR Guard: Verify ownership
	if booking.UserID != userID {
		return nil, errors.New("unauthorized: you do not own this booking")
	}

	// 3. State Guard: Prevent double-cancellation
	if booking.Status == "cancelled" {
		return nil, errors.New("booking is already cancelled")
	}
	if booking.Status != "confirmed" && booking.Status != "pending" {
		return nil, fmt.Errorf("cannot cancel booking with status '%s'", booking.Status)
	}

	// 4. Handle Pending Booking (Release Redis hold, no money involved)
	if booking.Status == "pending" {
		s.rdb.Del(ctx, "hold:slot:"+booking.SlotID.String())
		now := time.Now()
		if err := s.bookingRepo.CancelBookingTx(ctx, booking.ID, req.Reason, "user", booking.SlotID, nil); err != nil {
			return nil, errors.New("failed to cancel pending booking")
		}
		return &CancelBookingResponse{
			BookingID:          booking.ID,
			Status:             "cancelled",
			RefundAmount:       0.0,
			RefundPercentage:   0.0,
			CancellationReason: req.Reason,
			CancelledAt:        now,
		}, nil
	}

	// 5. Confirmed Booking: Fetch Space, Slot, and CancellationPolicy
	space, err := s.spaceRepo.FindBySpaceID(booking.SpaceID)
	if err != nil {
		return nil, errors.New("space details not found")
	}
	slot, err := s.spaceRepo.FindBySlotID(booking.SlotID)
	if err != nil {
		return nil, errors.New("slot details not found")
	}
	policy, err := s.venueRepo.FindCancellationPolicyByVenueID(space.VenueID)
	if err != nil {
		return nil, errors.New("cancellation policy not found for this venue")
	}

	// 6. Policy Calculation Engine
	now := time.Now()
	refundPercentage := 0.0

	if space.BookingType == "daily" {
		// Daily spaces: calculated in days
		daysRemaining := int(slot.Date.Sub(now.Truncate(24*time.Hour)).Hours() / 24)
		if daysRemaining < 0 {
			return nil, errors.New("cannot cancel a booking for a past date")
		}
		if daysRemaining >= policy.FullRefundDays {
			refundPercentage = policy.FullRefundPercent
		} else if daysRemaining >= policy.PartialRefundDays {
			refundPercentage = policy.PartialRefundPercent
		}
	} else {
		// Hourly spaces: calculated in hours
		slotDateTime := slot.Date
		if slot.StartTime != nil {
			if parsedTime, err := time.Parse("15:04", *slot.StartTime); err == nil {
				slotDateTime = time.Date(slot.Date.Year(), slot.Date.Month(), slot.Date.Day(),
					parsedTime.Hour(), parsedTime.Minute(), 0, 0, slot.Date.Location())
			}
		}
		hoursRemaining := slotDateTime.Sub(now).Hours()
		if hoursRemaining <= 0 {
			return nil, errors.New("cannot cancel an ongoing or past slot")
		}
		if hoursRemaining >= float64(policy.FullRefundHours) {
			refundPercentage = policy.FullRefundPercent
		} else if hoursRemaining >= float64(policy.PartialRefundHours) {
			refundPercentage = policy.PartialRefundPercent
		}
	}

	// 7. Calculate Refund Amount strictly on AmountPaid
	refundAmount := booking.AmountPaid * (refundPercentage / 100.0)

	var refundRecord *domain.Refund
	if refundAmount > 0 {
		refundRecord = &domain.Refund{
			BookingID: booking.ID,
			Amount:    refundAmount,
			Status:    "pending",
			Reason:    req.Reason,
		}
	}

	// 8. Execute Atomic Cancellation Transaction
	if err := s.bookingRepo.CancelBookingTx(ctx, booking.ID, req.Reason, "user", booking.SlotID, refundRecord); err != nil {
		return nil, errors.New("failed to cancel booking in database")
	}

	return &CancelBookingResponse{
		BookingID:          booking.ID,
		Status:             "cancelled",
		RefundAmount:       refundAmount,
		RefundPercentage:   refundPercentage,
		CancellationReason: req.Reason,
		CancelledAt:        now,
	}, nil
}
