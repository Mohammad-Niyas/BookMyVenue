package domain

import (
	"time"
	"github.com/google/uuid"
)

type Refund struct {
	ID               uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PaymentID        uuid.UUID `gorm:"type:uuid;not null"` 
	Payment          Payment   `gorm:"foreignKey:PaymentID"`
	
	RazorpayRefundID string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Amount           float64   `gorm:"type:decimal(10,2);not null"`
	Reason           string    `gorm:"type:text"` 
	Status           string    `gorm:"type:varchar(20);default:'pending';not null"`
	
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`
}