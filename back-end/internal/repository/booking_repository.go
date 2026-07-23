package repository

import (
	"bookmyvenue/internal/domain"
	"context"

	"gorm.io/gorm"
)

type BookingRepository interface{
	 Create(ctx context.Context, booking *domain.Booking) error
}

type bookingRepository struct{
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) BookingRepository {
	return &bookingRepository{db: db}
}

func (r *bookingRepository)  Create(ctx context.Context, booking *domain.Booking) error{
	return r.db.WithContext(ctx).Create(booking).Error
}



