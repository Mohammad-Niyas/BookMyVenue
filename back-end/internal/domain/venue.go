package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Venue struct {
	ID              uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OwnerID         uuid.UUID `gorm:"type:uuid;not null"`
	Owner           User      `gorm:"foreignKey:OwnerID"`
	Name            string    `gorm:"type:varchar(150);not null"`
	Description     string    `gorm:"type:text;not null"`
	Type            string    `gorm:"type:varchar(50);index;not null"`
	Address         string    `gorm:"type:text;not null"`
	City            string    `gorm:"type:varchar(100);index;not null"`
	Rules           string    `gorm:"type:text"`                               
	Timings         string    `gorm:"type:varchar(100)"`                          
	Status          string    `gorm:"type:varchar(20);default:'pending';not null"` 
	RejectionReason *string   `gorm:"type:text"`     
	// S3 image 
	Images    []string  `gorm:"serializer:json"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
	// Relationships
	Spaces             []Space             `gorm:"foreignKey:VenueID;constraint:OnDelete:CASCADE"`
	CancellationPolicy *CancellationPolicy `gorm:"foreignKey:VenueID;constraint:OnDelete:CASCADE"`
	VenueEditDrafts    []VenueEditDraft    `gorm:"foreignKey:VenueID;constraint:OnDelete:CASCADE"`
}