package messaging

import (
	"fmt"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

// RabbitMQConnection struct to hold RabbitMQ connection
type RabbitMQConnection struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	mu      sync.Mutex
}

// NewRabbitMQConnection function to create new RabbitMQ connection
func NewRabbitMQConnection() (*RabbitMQConnection, error) {
	rmq := &RabbitMQConnection{}
	err := rmq.connect()
	if err != nil {
		return nil, err
	}

	// Declare exchange here
	err = rmq.channel.ExchangeDeclare(
		"events_exchange", // Exchange name
		"topic",           // Exchange type
		true,              // Durable
		false,             // Auto-deleted
		false,             // Internal
		false,             // No wait
		nil,               // Arguments
	)
	if err != nil {
		logrus.Errorf("Failed to declare exchange during initialization: %v", err)
		return nil, err
	}

	return rmq, nil
}

// connect function to RabbitMQ
func (rmq *RabbitMQConnection) connect() error {
	rmq.mu.Lock()
	defer rmq.mu.Unlock()

	rabbitMQURI := os.Getenv("RABBITMQ_URI")
	var err error

	// Retry mechanism
	for i := 0; i < 3; i++ {
		rmq.conn, err = amqp091.Dial(rabbitMQURI)
		if err == nil {
			logrus.Info("Successfully connected to RabbitMQ")
			rmq.channel, err = rmq.conn.Channel()
			if err != nil {
				logrus.Errorf("Failed to open a channel: %v", err)
				return err
			}
			logrus.Info("Successfully opened a channel")
			return nil
		}

		logrus.Warnf("Failed to connect to RabbitMQ (attempt %d): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("unable to connect to RabbitMQ after retries")
}

// GetConnection function to get RabbitMQ connection
func (rmq *RabbitMQConnection) GetConnection() (*amqp091.Connection, *amqp091.Channel, error) {
	rmq.mu.Lock()
	defer rmq.mu.Unlock()

	// Check if the connection or channel is closed
	if rmq.conn == nil || rmq.conn.IsClosed() {
		logrus.Warn("Reconnecting to RabbitMQ...")
		err := rmq.connect()
		if err != nil {
			return nil, nil, err
		}
	}

	// If the channel is nil or closed, open a new one
	if rmq.channel == nil || rmq.channel.IsClosed() {
		logrus.Warn("Reopening RabbitMQ channel...")
		ch, err := rmq.conn.Channel()
		if err != nil {
			return nil, nil, err
		}
		rmq.channel = ch
	}

	return rmq.conn, rmq.channel, nil
}

// Close function to close RabbitMQ connection
func (rmq *RabbitMQConnection) Close() {
	rmq.mu.Lock()
	defer rmq.mu.Unlock()

	if rmq.conn != nil {
		logrus.Info("Closing RabbitMQ connection...")
		rmq.conn.Close()
	}
}

// PublishEvent sends an event to RabbitMQ
func (rmq *RabbitMQConnection) PublishEvent(eventName string, body []byte) error {
	_, ch, err := rmq.GetConnection()
	if err != nil {
		logrus.Errorf("RabbitMQ not initialized: %v", err)
		return err
	}

	// Declare exchange for event if not already declared
	err = ch.ExchangeDeclare(
		"events_exchange", // Exchange name
		"topic",           // Exchange type
		true,              // Durable (persists even if RabbitMQ restarts)
		false,             // Auto-deleted (delete when no queues are bound)
		false,             // Internal (used internally by RabbitMQ)
		false,             // No wait
		nil,               // Arguments
	)
	if err != nil {
		logrus.Errorf("Failed to declare exchange: %v", err)
		return err
	}

	for i := 0; i < 3; i++ {
		logrus.Infof("[RabbitMQ] SENDING EVENT: %s | BODY: %s", eventName, body)
		err = ch.Publish(
			"events_exchange",
			eventName,
			false,
			false,
			amqp091.Publishing{
				ContentType: "application/json",
				Body:        body,
			},
		)

		if err == nil {
			logrus.Infof("Published event: %s | Body: %s", eventName, body)
			return nil
		}

		logrus.Warnf("Failed to publish event (attempt %d): %v", i+1, err)
		time.Sleep(1 * time.Second)
	}

	logrus.Errorf("Final failure: Could not publish event after retries")
	return err
}
