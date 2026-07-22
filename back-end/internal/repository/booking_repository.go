package repository

import (
	"bookmyvenue/internal/domain"

	"gorm.io/gorm"
)

type BookingRepository interface{
	Create(booking *domain.Booking) error
}

type bookingRepository struct{
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) BookingRepository {
	return &bookingRepository{db: db}
}

func (r *bookingRepository) Create(booking *domain.Booking) error{
	return r.db.Create(booking).Error
}



