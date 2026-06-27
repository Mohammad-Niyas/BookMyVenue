package domain

import (
	"time"
	"github.com/google/uuid"
)

type OutboxEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Topic     string    `gorm:"type:varchar(100);not null;index"`                
	Payload   string    `gorm:"type:text;not null"`
	Status    string    `gorm:"type:varchar(20);default:'pending';not null;index"`
	Retries   int       `gorm:"type:integer;default:0;not null"`
	ErrorLog  *string   `gorm:"type:text"`
	
	CreatedAt time.Time `gorm:"autoCreateTime;index"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}