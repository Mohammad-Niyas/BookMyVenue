package domain

import (
	"time"
	"github.com/google/uuid"
)

type PaymentAuditLog struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PaymentID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Payment    Payment   `gorm:"foreignKey:PaymentID"`

	FromStatus string    `gorm:"type:varchar(20);not null"`
	ToStatus   string    `gorm:"type:varchar(20);not null"`
	Metadata   string    `gorm:"type:text"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}