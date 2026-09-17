package service

import (
    "bookmyvenue/pkg/rabbitmq"
    "context"
    "log"
)

type NotificationConsumer interface {
    Start(ctx context.Context)
}

type notificationConsumer struct {
    rabbitMQ *rabbitmq.RabbitMQ
}

func NewNotificationConsumer(rabbitMQ *rabbitmq.RabbitMQ) NotificationConsumer {
    return &notificationConsumer{rabbitMQ: rabbitMQ}
}

func (c *notificationConsumer) Start(ctx context.Context) {
    _, err := c.rabbitMQ.DeclareQueue("booking.notifications")
    if err != nil {
        log.Printf("[Notification Consumer] Failed to declare queue: %v", err)
        return
    }

    msgs, err := c.rabbitMQ.Consume("booking.notifications", "notification_worker")
    if err != nil {
        log.Printf("[Notification Consumer] Failed to start consuming: %v", err)
        return
    }

    log.Println("[Notification Consumer] Listening for booking notifications...")

    for {
        select {
        case <-ctx.Done():
            log.Println("[Notification Consumer] Context cancelled, stopping listener")
            return
        case msg, ok := <-msgs:
            if !ok {
                log.Println("[Notification Consumer] Message delivery channel closed")
                return
            }

            log.Printf("[Notification Consumer] Booking Notification Dispatched! Payload: %s", string(msg.Body))

            if err := msg.Ack(false); err != nil {
                log.Printf("[Notification Consumer] Failed to ACK message: %v", err)
            }
        }
    }
}