package domain

import (
	"time"
	"github.com/google/uuid"
)

type Payment struct {
	ID                uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	BookingID         uuid.UUID  `gorm:"type:uuid;not null"` 
	Booking           Booking    `gorm:"foreignKey:BookingID"` 
	
	RazorpayOrderID   string     `gorm:"type:varchar(100);uniqueIndex;not null"`
	RazorpayPaymentID *string    `gorm:"type:varchar(100)"`
	RazorpaySignature *string    `gorm:"type:varchar(255)"` 
	
	Amount            float64    `gorm:"type:decimal(10,2);not null"`
	Status            string     `gorm:"type:varchar(20);default:'pending';not null"` 
	
	CreatedAt         time.Time  `gorm:"autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime"`

	// Relationships
	AuditLogs         []PaymentAuditLog `gorm:"foreignKey:PaymentID;constraint:OnDelete:CASCADE"`
}