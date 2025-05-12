package queue

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-redis/redis"
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"
	"ussd-wrapper/connections"
	"ussd-wrapper/library/logger"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"ussd-wrapper/constants"
)

// Manager manages multiple RabbitMQ consumers
type Manager struct {
	DB           *sql.DB
	DBSlave      *sql.DB
	Tracer       trace.Tracer
	RedisConn    *redis.Client
	RabbitClient *connections.RabbitMQClient

	TotalConfigured int32 // total intended queues
	TotalStarted    int32 // atomic counter
}

// NewQueueManager creates a new queue manager instance
func NewQueueManager(tracer trace.Tracer, db *sql.DB, dbSlave *sql.DB, redis *redis.Client) (*Manager, error) {
	// Get the RabbitMQ client
	rabbitClient := connections.GetClient()
	if rabbitClient == nil {
		return nil, fmt.Errorf("failed to get RabbitMQ client")
	}

	return &Manager{
		DB:           db,
		DBSlave:      dbSlave,
		Tracer:       tracer,
		RedisConn:    redis,
		RabbitClient: rabbitClient,
	}, nil
}

// InitializeQueues sets up all configured consumer queues
func (qm *Manager) InitializeQueues(ctx context.Context) {
	ctx, span := qm.Tracer.Start(ctx, "InitQueues")
	defer span.End()

	// Get configuration from environment
	queues := os.Getenv("queues")
	channels := os.Getenv("channels")

	// Get all the queues to consume from
	parts := strings.Split(queues, ",")
	sort.Strings(parts)

	partsChannels := strings.Split(channels, ",")
	sort.Strings(partsChannels)

	qm.TotalConfigured = int32(len(parts) * len(partsChannels)) // total expected queues

	for _, table := range partsChannels {
		channel := strings.ToLower(strings.TrimSpace(table))

		for _, action := range parts {
			action = strings.ToLower(strings.TrimSpace(action))
			queueName := fmt.Sprintf("%s.%s", channel, action)

			go qm.SetupQueue(ctx, queueName)
		}
	}

	// Block forever to keep consumers running
	select {}
}

// SetupQueue configures and starts a consumer for a specific queue
func (qm *Manager) SetupQueue(ctx context.Context, queueName string) {
	ctx, span := qm.Tracer.Start(ctx, "SetupQueue",
		trace.WithAttributes(attribute.String("queueName", queueName)))
	defer span.End()

	// Create a unique consumer name for each queue
	consumerName := fmt.Sprintf("consumer-%s", queueName)

	// Start consuming messages
	deliveries, err := qm.RabbitClient.Consume(ctx, queueName, consumerName)
	if err != nil {
		logger.WithCtx(ctx).WithFields(
			logrus.Fields{
				constants.DESCRIPTION: "error setting up consumer",
				constants.DATA:        queueName,
			}).Panic()
	}

	// Update started counter
	started := atomic.AddInt32(&qm.TotalStarted, 1)

	logger.WithCtx(ctx).
		WithFields(logrus.Fields{
			"queue":         queueName,
			"started_total": started,
			"of_configured": qm.TotalConfigured,
		}).
		Info("✅ Queue started")

	// Start processing messages
	go qm.processDeliveries(ctx, deliveries, queueName)

	// Block this goroutine
	forever := make(chan bool)
	<-forever
}

// processDeliveries handles incoming messages from a queue
func (qm *Manager) processDeliveries(ctx context.Context, deliveries <-chan amqp.Delivery, queueName string) {
	for {
		select {
		case <-ctx.Done():
			logrus.WithContext(ctx).
				WithFields(logrus.Fields{
					"queue": queueName,
				}).
				Info("Consumer shutdown requested")
			return

		case delivery, ok := <-deliveries:
			if !ok {
				logger.WithCtx(ctx).
					WithFields(logrus.Fields{
						"queue": queueName,
					}).
					Warn("Delivery channel closed, attempting to reconnect")

				// Channel was closed, attempt to reconnect
				newCtx := context.Background() // Create a new context since the old one might be canceled
				newDeliveries, err := qm.RabbitClient.Consume(newCtx, queueName, fmt.Sprintf("consumer-%s", queueName))
				if err != nil {
					logger.WithCtx(newCtx).
						WithFields(logrus.Fields{
							constants.DESCRIPTION: "failed to reconnect consumer",
							constants.DATA:        queueName,
						}).Error(err.Error())

					// Wait a bit before trying again
					time.Sleep(5 * time.Second)
					continue
				}

				// Replace deliveries channel and continue
				deliveries = newDeliveries
				continue
			}

			// Process the delivery
			err := qm.RouteMessage(ctx, delivery, queueName)
			if err != nil {
				logger.WithCtx(ctx).
					WithFields(logrus.Fields{
						constants.DESCRIPTION: "error processing message",
						constants.DATA:        queueName,
						"error":               err.Error(),
					}).Error("Failed to process message")

				// Nack the message to requeue it
				if err := delivery.Nack(false, true); err != nil {
					logger.WithCtx(ctx).
						WithFields(logrus.Fields{
							constants.DESCRIPTION: "failed to nack message",
							constants.DATA:        queueName,
						}).Error(err.Error())
				}
			} else {
				// Ack the message
				if err := delivery.Ack(false); err != nil {
					logger.WithCtx(ctx).
						WithFields(logrus.Fields{
							constants.DESCRIPTION: "failed to ack message",
							constants.DATA:        queueName,
						}).Error(err.Error())
				}
			}
		}
	}
}

// RouteMessage determines how to process a message based on the queue name
func (qm *Manager) RouteMessage(ctx context.Context, delivery amqp.Delivery, queue string) error {
	ctx, span := qm.Tracer.Start(ctx, "RouteMessage",
		trace.WithAttributes(attribute.String("queueName", queue)))
	defer span.End()

	prefix := os.Getenv("queue_prefix")

	if len(prefix) > 0 {
		parts := strings.Split(queue, ".")
		if len(parts) > 1 {
			queue = strings.Join(parts[1:], ".")
		}
	}

	queue = strings.ToLower(queue)

	// Create a slice with a single delivery to maintain compatibility with existing code
	deliveries := make(chan amqp.Delivery, 1)
	deliveries <- delivery
	close(deliveries)

	if queue == "bet_settlement" {
		return qm.ProcessSettlement(ctx, deliveries, queue)
	} else if strings.HasPrefix(queue, "bet_settlement") {
		return qm.ProcessSettlement(ctx, deliveries, queue)
	} else if queue == "rollback_bet_settlement" {
		return qm.ProcessSettlementRollback(ctx, deliveries)
	} else if queue == "bet_settlement_rollback" {
		return qm.ProcessSettlementRollback(ctx, deliveries)
	} else if queue == "bet_closure" {
		return qm.ProcessBetClosure(ctx, deliveries)
	} else if queue == "bet_approval" {
		return qm.ProcessBetApproval(ctx, deliveries)
	}

	return nil
}

// PublishMessage sends a message to a RabbitMQ queue
func (qm *Manager) PublishMessage(ctx context.Context, queueName string, payload interface{}, priority uint8) error {
	ctx, span := qm.Tracer.Start(ctx, "PublishMessage",
		trace.WithAttributes(attribute.String("queueName", queueName)))
	defer span.End()

	err := qm.RabbitClient.Publish(ctx, queueName, payload, priority)
	if err != nil {
		return fmt.Errorf("failed to publish message to queue %s: %w", queueName, err)
	}

	return nil
}

// ProcessSettlement handles bet settlement messages
func (qm *Manager) ProcessSettlement(ctx context.Context, deliveries <-chan amqp.Delivery, queue string) error {
	// Implementation would go here
	return nil
}

// ProcessSettlementRollback handles bet settlement rollback messages
func (qm *Manager) ProcessSettlementRollback(ctx context.Context, deliveries <-chan amqp.Delivery) error {
	// Implementation would go here
	return nil
}

// ProcessBetClosure handles bet closure messages
func (qm *Manager) ProcessBetClosure(ctx context.Context, deliveries <-chan amqp.Delivery) error {
	// Implementation would go here
	return nil
}

// ProcessBetApproval handles bet approval messages
func (qm *Manager) ProcessBetApproval(ctx context.Context, deliveries <-chan amqp.Delivery) error {
	// Implementation would go here
	return nil
}
