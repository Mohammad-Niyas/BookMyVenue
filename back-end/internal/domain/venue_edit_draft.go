package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VenueEditDraft struct {
	ID              uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	VenueID         uuid.UUID `gorm:"type:uuid;not null;index"`
	Venue           Venue     `gorm:"foreignKey:VenueID;constraint:OnDelete:CASCADE"`

	Name            string    `gorm:"type:varchar(150);not null"`
	Description     string    `gorm:"type:text;not null"`
	Type            string    `gorm:"type:varchar(50);not null"`
	Address         string    `gorm:"type:text;not null"`
	City            string    `gorm:"type:varchar(100);not null"`
	Rules           string    `gorm:"type:text"`
	Timings         string    `gorm:"type:varchar(100)"`
	Images          []string  `gorm:"serializer:json"`

	Status          string    `gorm:"type:varchar(20);default:'pending_review';not null;index"`
	AdminNote       *string   `gorm:"type:text"`

	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}