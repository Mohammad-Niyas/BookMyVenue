package repository

import (
	"bookmyvenue/internal/domain"
	"context"
  "errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentRepository interface {
	CreateOrder(payment *domain.Payment) error
	FindByBookingID(bookingID uuid.UUID) (*domain.Payment, error)
	FindByRazorpayOrderID(orderID string) (*domain.Payment, error)
	ConfirmBookingAndCapturePayment(ctx context.Context, paymentID uuid.UUID, rzpPaymentID string, signature string, bookingID uuid.UUID, amountPaid float64, slotID uuid.UUID) error
	CreateAuditLog(audit *domain.PaymentAuditLog) error
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) CreateOrder(payment *domain.Payment) error {
	return r.db.Create(payment).Error
}

func (r *paymentRepository) FindByBookingID(bookingID uuid.UUID) (*domain.Payment, error) {
	var payment domain.Payment
	err := r.db.Where("booking_id = ?", bookingID).First(&payment).Error
	return &payment, err
}

func (r *paymentRepository) FindByRazorpayOrderID(orderID string) (*domain.Payment, error) {
	var payment domain.Payment
	err := r.db.Where("razorpay_order_id = ?", orderID).First(&payment).Error
	return &payment, err
}

func (r *paymentRepository) ConfirmBookingAndCapturePayment(ctx context.Context, paymentID uuid.UUID, rzpPaymentID string, signature string, bookingID uuid.UUID, amountPaid float64, slotID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		paymentUpdates := map[string]interface{}{
			"status":              "captured",
			"razorpay_payment_id": rzpPaymentID,
			"razorpay_signature":  signature,
		}
		if err := tx.Model(&domain.Payment{}).Where("id = ?", paymentID).Updates(paymentUpdates).Error; err != nil {
			return err
		}

		bookingUpdates := map[string]interface{}{
			"status":      "confirmed",
			"amount_paid": amountPaid,
		}
		if err := tx.Model(&domain.Booking{}).Where("id = ?", bookingID).Updates(bookingUpdates).Error; err != nil {
			return err
		}

		// Atomic Conditional Update: Update ONLY if is_booked is still false
		slotRes := tx.Model(&domain.Slot{}).
			Where("id = ? AND is_booked = false", slotID).
			Update("is_booked", true)

		if slotRes.Error != nil {
			return slotRes.Error
		}

		// If RowsAffected is 0, someone already booked this slot while this user was delayed!
		if slotRes.RowsAffected == 0 {
			return errors.New("slot_already_booked_conflict")
		}

		auditLog := &domain.PaymentAuditLog{
			PaymentID:  paymentID,
			FromStatus: "created",
			ToStatus:   "captured",
			Metadata:   "payment verified via HMAC-SHA256 signature and booking confirmed",
		}
		if err := tx.Create(auditLog).Error; err != nil {
			return err
		}
		payload := fmt.Sprintf(`{"booking_id":"%s","payment_id":"%s","amount_paid":%.2f}`, bookingID, paymentID, amountPaid)
		outboxEvent := &domain.OutboxEvent{
			Topic:   "booking.confirmed",
			Payload: payload,
			Status:  "pending",
		}
		if err := tx.Create(outboxEvent).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *paymentRepository) CreateAuditLog(audit *domain.PaymentAuditLog) error {
	return r.db.Create(audit).Error
}

2) booking_service.go:

package service

import (
	"bookmyvenue/internal/domain"
	"bookmyvenue/internal/repository"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
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

type BookingService interface {
	CreateBooking(ctx context.Context, userID uuid.UUID, req BookingRequest) (*BookingResponse, error)
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
  
		if space.BookingType == "daily" && space.Capacity >= 200 {
		minBookDate := time.Now().AddDate(0, 0, 30).Truncate(24 * time.Hour)
		if slot.Date.Before(minBookDate) {
			return nil, errors.New("daily venues (auditoriums/banquet halls) must be booked at least 30 days in advance")
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
		s.rdb.Del(ctx, redisKey)
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