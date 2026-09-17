package rabbitmq

import (
	"context"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// NewRabbitMQ establishes a TCP connection and opens a multiplexed channel.
func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	log.Println("[RabbitMQ] Connected and channel opened successfully")
	return &RabbitMQ{conn: conn, ch: ch}, nil
}

// DeclareQueue creates a durable queue if it does not already exist.
func (r *RabbitMQ) DeclareQueue(name string) (amqp.Queue, error) {
	return r.ch.QueueDeclare(
		name,
		true,
		false,
		false,
		false,
		nil,
	)
}

// Publish sends a persistent JSON payload to an exchange with a routing key.
func (r *RabbitMQ) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	return r.ch.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)
}

// Consume starts a consumer that receives delivery frames with manual ACK required.
func (r *RabbitMQ) Consume(queueName, consumerTag string) (<-chan amqp.Delivery, error) {
	return r.ch.Consume(
		queueName,
		consumerTag,
		false,
		false,
		false,
		false,
		nil,
	)
}

// Close gracefully terminates channel and connection.
func (r *RabbitMQ) Close() error {
	if r.ch != nil {
		_ = r.ch.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}