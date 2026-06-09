	package domain

	import (
		"time"
		"github.com/google/uuid"
	)

	type User struct {
		ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
		Name         string    `gorm:"type:varchar(100);not null"`
		Email        string    `gorm:"type:varchar(255);uniqueIndex;not null"`
		PasswordHash string    `gorm:"type:varchar(255);not null"`
		Role         string    `gorm:"type:varchar(20);default:'user';not null"` // Allowed: 'user', 'owner'
		
		// Owner Specific Fields
		Phone        *string   `gorm:"type:varchar(15)"`
		BusinessName *string   `gorm:"type:varchar(150)"`
		GSTNumber    *string   `gorm:"type:varchar(15)"`
		
		CreatedAt    time.Time `gorm:"autoCreateTime"`
		UpdatedAt    time.Time `gorm:"autoUpdateTime"`
	}