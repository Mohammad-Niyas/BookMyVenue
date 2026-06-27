package domain

import (
	"time"
	"github.com/google/uuid"
)

type Booking struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	
	// User Relation
	UserID      uuid.UUID `gorm:"type:uuid;not null"` 
	User        User      `gorm:"foreignKey:UserID"`
	
	// Space Relation
	SpaceID     uuid.UUID `gorm:"type:uuid;not null"` 
	Space       Space     `gorm:"foreignKey:SpaceID"` 
	
	// Slot Relation
	SlotID      uuid.UUID `gorm:"type:uuid;not null"` 
	Slot        Slot      `gorm:"foreignKey:SlotID"` 
	
	TotalAmount float64   `gorm:"type:decimal(10,2);not null"`
	AmountPaid  float64   `gorm:"type:decimal(10,2);default:0.00;not null"`
	Status      string    `gorm:"type:varchar(20);default:'pending';not null"` 
	CancellationReason *string    `gorm:"type:text"`
	CancelledBy        *string    `gorm:"type:varchar(20)"`
	CancelledAt        *time.Time `gorm:"type:timestamp"`

	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}