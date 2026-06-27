package domain

import (
	"time"
	"github.com/google/uuid"
)

type Space struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	
	// Venue Relation
	VenueID     uuid.UUID `gorm:"type:uuid;not null"` 
	Venue       Venue     `gorm:"foreignKey:VenueID"`
	
	Name        string    `gorm:"type:varchar(100);not null"` 
	Capacity    int       `gorm:"not null"`
	Price       float64   `gorm:"type:decimal(10,2);not null"`
	BookingType string    `gorm:"type:varchar(20);default:'daily';not null"` 
	Images      []string  `gorm:"serializer:json"` 
	
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
	
	Slots       []Slot    `gorm:"foreignKey:SpaceID;constraint:OnDelete:CASCADE"`
}