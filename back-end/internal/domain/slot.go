package domain

import (
	"time"
	"github.com/google/uuid"
)

type Slot struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SpaceID   uuid.UUID  `gorm:"type:uuid;not null;index"` 
	Space     Space      `gorm:"foreignKey:SpaceID"` 
	
	Date      time.Time  `gorm:"type:date;not null;index"`
	IsBooked  bool       `gorm:"default:false;not null"`
	BookingID *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL"` 
	
	StartTime *string    `gorm:"type:varchar(5)"` 
	EndTime   *string    `gorm:"type:varchar(5)"` 

	CreatedAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime"`
}