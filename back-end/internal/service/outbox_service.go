package service

import (
	"bookmyvenue/internal/domain"
	"bookmyvenue/pkg/rabbitmq"
	"context"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OutboxService interface {
	Start(ctx context.Context)
}

type outboxService struct {
	db       *gorm.DB
	rabbitMQ *rabbitmq.RabbitMQ
	interval time.Duration
}

func NewOutboxService(db *gorm.DB, rabbitMQ *rabbitmq.RabbitMQ, interval time.Duration) OutboxService {
	return &outboxService{
		db:       db,
		rabbitMQ: rabbitMQ,
		interval: interval,
	}
}

func (s *outboxService) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Println("[Outbox Worker] Started polling background events...")

	for {
		select {
		case <-ctx.Done():
			log.Println("[Outbox Worker] Context cancelled, shutting down gracefully")
			return
		case <-ticker.C:
			s.processBatch(ctx)
		}
	}
}

func (s *outboxService) processBatch(ctx context.Context) {
	var events []domain.OutboxEvent

	// Open a database transaction with context
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		log.Printf("[Outbox Worker] Failed to start transaction: %v", tx.Error)
		return
	}
	defer tx.Rollback()

	// Query pending events with FOR UPDATE SKIP LOCKED
	err := tx.Clauses(clause.Locking{
		Strength: "UPDATE",
		Options:  "SKIP LOCKED",
	}).Where("status = ?", "pending").
		Order("created_at ASC").
		Limit(20).
		Find(&events).Error

	if err != nil {
		log.Printf("[Outbox Worker] Query failed: %v", err)
		return
	}

	if len(events) == 0 {
		return
	}

	// Iterate and publish to RabbitMQ
	for i := range events {
		event := &events[i]

		pubErr := s.rabbitMQ.Publish(ctx, "", "booking.notifications", []byte(event.Payload))
		if pubErr != nil {
			event.Retries++
			errMsg := pubErr.Error()
			event.ErrorLog = &errMsg

			if event.Retries >= 5 {
				event.Status = "failed"
				log.Printf("[Outbox Worker] Event %s failed after 5 retries: %v", event.ID, pubErr)
			}
		} else {
			event.Status = "published"
			now := time.Now()
			event.UpdatedAt = now
		}

		if saveErr := tx.Save(event).Error; saveErr != nil {
			log.Printf("[Outbox Worker] Failed to update event %s status: %v", event.ID, saveErr)
		}
	}

	// Commit transaction to release row locks
	if commitErr := tx.Commit().Error; commitErr != nil {
		log.Printf("[Outbox Worker] Commit failed: %v", commitErr)
	}
}