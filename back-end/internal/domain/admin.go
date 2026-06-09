package domain

import (
	"time"
	"github.com/google/uuid"
)

type Admin struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name         string    `gorm:"type:varchar(100);not null"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string    `gorm:"type:varchar(255);not null"`
	
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}