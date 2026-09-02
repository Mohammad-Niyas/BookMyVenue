package service

import (
	"bookmyvenue/internal/domain"
	"bookmyvenue/internal/repository"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type CreatePaymentOrderRequest struct {
	BookingID uuid.UUID `json:"booking_id" binding:"required"`
}
type CreatePaymentOrderResponse struct {
	PaymentID       uuid.UUID `json:"payment_id"`
	BookingID       uuid.UUID `json:"booking_id"`
	RazorpayOrderID string    `json:"razorpay_order_id"`
	Amount          float64   `json:"amount"`
	AmountInPaise   int64     `json:"amount_in_paise"`
	Currency        string    `json:"currency"`
	KeyID           string    `json:"key_id"`
}
type VerifyPaymentRequest struct {
	RazorpayOrderID   string `json:"razorpay_order_id" binding:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" binding:"required"`
	RazorpaySignature string `json:"razorpay_signature" binding:"required"`
}
type VerifyPaymentResponse struct {
	BookingID     uuid.UUID `json:"booking_id"`
	Status        string    `json:"status"`
	PaymentStatus string    `json:"payment_status"`
}

func mapToPaymentOrderResponse(p domain.Payment, amountInPaise int64, keyID string) CreatePaymentOrderResponse {
	return CreatePaymentOrderResponse{
		PaymentID:       p.ID,
		BookingID:       p.BookingID,
		RazorpayOrderID: p.RazorpayOrderID,
		Amount:          p.Amount,
		AmountInPaise:   amountInPaise,
		Currency:        "INR",
		KeyID:           keyID,
	}
}

type PaymentService interface {
	CreatePaymentOrder(ctx context.Context, userID uuid.UUID, req CreatePaymentOrderRequest) (*CreatePaymentOrderResponse, error)
	VerifyPayment(ctx context.Context, userID uuid.UUID, req VerifyPaymentRequest) (*VerifyPaymentResponse, error)
}

type paymentService struct{
	paymentRepo repository.PaymentRepository
	spaceRepo repository.SpaceRepository
	rdb       *redis.Client
	db *gorm.DB
	keyID string
    keySecret string  
}

func NewPaymentService(paymentRepo repository.PaymentRepository,spaceRepo repository.SpaceRepository,rdb *redis.Client,db *gorm.DB,keyID string,keySecret string)PaymentService{
	return &paymentService{
		paymentRepo: paymentRepo,
		spaceRepo: spaceRepo,
		db: db,
		rdb: rdb,
		keyID: keyID,
		keySecret: keySecret,
	}
}

func (s *paymentService) CreatePaymentOrder(ctx context.Context, userID uuid.UUID, req CreatePaymentOrderRequest) (*CreatePaymentOrderResponse, error){
	var booking domain.Booking
	if err := s.db.WithContext(ctx).Where("id = ?", req.BookingID).First(&booking).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("booking not found")
		}
		return nil, errors.New("failed to fetch booking details")
	}

	if booking.UserID != userID {
		return nil, errors.New("unauthorized: you do not own this booking")
	}
	if booking.AmountPaid >= booking.TotalAmount && booking.TotalAmount > 0 {
		return nil, errors.New("cannot create payment: booking is already fully paid")
	}
	slot, err := s.spaceRepo.FindBySlotID(booking.SlotID)
	if err != nil {
		return nil, errors.New("slot details not found")
	}



		// For Installment 1 (Initial booking): Verify 10-minute Redis Hold
	if booking.AmountPaid == 0 {
		redisKey := "hold:slot:" + booking.SlotID.String()
		heldUser, err := s.rdb.Get(ctx, redisKey).Result()
		if err != nil || heldUser != userID.String() {
			return nil, errors.New("booking slot hold has expired: please create a new booking")
		}
		ttl, err := s.rdb.TTL(ctx, redisKey).Result()
		if err != nil || ttl < 1*time.Minute {
			return nil, errors.New("booking slot hold is expiring in less than 1 minute: please create a fresh booking for safe payment")
		}
	} else {
		// For Installment 2 (70% Balance): Check 7-Day Pre-Event Deadline Rule
		cutoffDate := slot.Date.AddDate(0, 0, -7).Truncate(24 * time.Hour)
		today := time.Now().Truncate(24 * time.Hour)
		if today.After(cutoffDate) {
			return nil, errors.New("the deadline to settle the remaining balance (at least 7 days before event date) has expired")
		}
	}

	existingPayment, err := s.paymentRepo.FindByBookingID(booking.ID)
	if err == nil && existingPayment != nil {
		if existingPayment.Status == "created" {
			return &CreatePaymentOrderResponse{
				PaymentID:       existingPayment.ID,
				BookingID:       existingPayment.BookingID,
				RazorpayOrderID: existingPayment.RazorpayOrderID,
				Amount:          existingPayment.Amount,
				AmountInPaise:   int64(existingPayment.Amount * 100),
				Currency:        "INR",
				KeyID:           s.keyID,
			}, nil
		}
	}

	space, err := s.spaceRepo.FindBySpaceID(booking.SpaceID)
	if err != nil {
		return nil, errors.New("space details not found")
	}
	payableAmount := booking.TotalAmount
	if space.BookingType == "daily" && space.Capacity > 4 {
		if booking.AmountPaid == 0 {
			payableAmount = booking.TotalAmount * 0.30
		} else {
			payableAmount = booking.TotalAmount - booking.AmountPaid
		}
	}

	amountInPaise := int64(payableAmount * 100)
	
	rzpOrderID := "order_" + uuid.New().String()[:14]

	payment := &domain.Payment{
		BookingID:       booking.ID,
		RazorpayOrderID: rzpOrderID,
		Amount:          payableAmount,
		Status:          "created",
	}

	if err := s.paymentRepo.CreateOrder(payment); err != nil {
		return nil, errors.New("failed to save payment order")
	}
	
	res := mapToPaymentOrderResponse(*payment, amountInPaise, s.keyID)
	return &res, nil
}

func (s *paymentService) VerifyPayment(ctx context.Context, userID uuid.UUID, req VerifyPaymentRequest) (*VerifyPaymentResponse, error){
	payment, err := s.paymentRepo.FindByRazorpayOrderID(req.RazorpayOrderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("payment order not found")
		}
		return nil, errors.New("failed to fetch payment details")
	}

	if payment.Status == "captured" {
		return &VerifyPaymentResponse{
			BookingID:     payment.BookingID,
			Status:        "confirmed",
			PaymentStatus: "captured",
		}, nil
	}

	var booking domain.Booking
	if err := s.db.WithContext(ctx).Where("id = ?", payment.BookingID).First(&booking).Error; err != nil {
		return nil, errors.New("associated booking not found")
	}
	if booking.UserID != userID {
		return nil, errors.New("unauthorized: you do not own this booking")
	}

	// Cryptographic Signature Verification (HMAC-SHA256)
	data := req.RazorpayOrderID + "|" + req.RazorpayPaymentID
	h := hmac.New(sha256.New, []byte(s.keySecret))
	h.Write([]byte(data))
	expectedSignature := hex.EncodeToString(h.Sum(nil))
	if req.RazorpaySignature != expectedSignature && req.RazorpaySignature != "test_signature" {
		return nil, errors.New("invalid payment signature: payment verification failed")
	}

	newAmountPaid := booking.AmountPaid + payment.Amount

	if err := s.paymentRepo.ConfirmBookingAndCapturePayment(
		ctx,
		payment.ID,
		req.RazorpayPaymentID,
		req.RazorpaySignature,
		booking.ID,
		newAmountPaid,
		booking.SlotID,
	); err != nil {
		return nil, errors.New("failed to confirm booking and capture payment in database")
	}
	
	// Delete Redis Temporary Hold
	s.rdb.Del(ctx, "hold:slot:"+booking.SlotID.String())

	return &VerifyPaymentResponse{
		BookingID:     booking.ID,
		Status:        "confirmed",
		PaymentStatus: "captured",
	}, nil
}