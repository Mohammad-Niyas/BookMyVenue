package repository

import (
	"bookmyvenue/internal/domain"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BookingRepository interface {
	Create(ctx context.Context, tx *gorm.DB, booking *domain.Booking) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error)
	CancelBookingTx(ctx context.Context, bookingID uuid.UUID, reason string, cancelledBy string, slotID uuid.UUID, refund *domain.Refund) error
}

type bookingRepository struct{
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) BookingRepository {
	return &bookingRepository{db: db}
}

func (r *bookingRepository)  Create(ctx context.Context,tx *gorm.DB, booking *domain.Booking) error{
	if tx != nil {
    	return tx.WithContext(ctx).Create(booking).Error
	}
	return r.db.WithContext(ctx).Create(booking).Error
}

func (r *bookingRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	var booking domain.Booking
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *bookingRepository) CancelBookingTx(ctx context.Context, bookingID uuid.UUID, reason string, cancelledBy string, slotID uuid.UUID, refund *domain.Refund) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		// 1. Update Booking record
		updates := map[string]interface{}{
			"status":              "cancelled",
			"cancellation_reason": reason,
			"cancelled_by":        cancelledBy,
			"cancelled_at":        &now,
		}
		if err := tx.Model(&domain.Booking{}).Where("id = ?", bookingID).Updates(updates).Error; err != nil {
			return err
		}
		// 2. Free the Slot so others can book it
		slotUpdates := map[string]interface{}{
			"is_booked":  false,
			"booking_id": nil,
		}
		if err := tx.Model(&domain.Slot{}).Where("id = ?", slotID).Updates(slotUpdates).Error; err != nil {
			return err
		}
		// 3. Create Refund Record (if refund is applicable)
		refundAmount := 0.0
		if refund != nil {
			refundAmount = refund.Amount
			if err := tx.Create(refund).Error; err != nil {
				return err
			}
		}
		// 4. Create Outbox Event for async notification (RabbitMQ)
		payload := fmt.Sprintf(`{"booking_id":"%s","refund_amount":%.2f,"cancelled_by":"%s"}`, bookingID, refundAmount, cancelledBy)
		outboxEvent := &domain.OutboxEvent{
			Topic:   "booking.cancelled",
			Payload: payload,
			Status:  "pending",
		}
		return tx.Create(outboxEvent).Error
	})
}



