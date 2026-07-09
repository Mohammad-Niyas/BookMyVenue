package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CancellationPolicy struct {
	ID                   uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	VenueID              uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	FullRefundDays       int       `gorm:"default:15;not null"`
	FullRefundPercent    float64   `gorm:"type:decimal(5,2);default:95.00;not null"` 
	
	PartialRefundDays    int       `gorm:"default:7;not null"`
	PartialRefundPercent float64   `gorm:"type:decimal(5,2);default:50.00;not null"`
	
	CreatedAt            time.Time `gorm:"autoCreateTime"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}