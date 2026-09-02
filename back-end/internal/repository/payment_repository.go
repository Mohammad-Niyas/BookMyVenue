package repository

import (
	"bookmyvenue/internal/domain"
	"context"

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
		if err := tx.Model(&domain.Slot{}).Where("id = ?", slotID).Update("is_booked", true).Error; err != nil {
			return err
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
		return nil
	})
}

func (r *paymentRepository) CreateAuditLog(audit *domain.PaymentAuditLog) error{
	return r.db.Create(audit).Error
}
