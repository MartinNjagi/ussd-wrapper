package connections

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"ussd-wrapper/library/logger"
)

// RabbitMQClient provides a unified interface for both publishing and consuming
type RabbitMQClient struct {
	conn       *amqp.Connection
	err        chan error
	mu         sync.Mutex
	reconnects int
	maxRetries int
	connected  bool

	// For tracking channels
	channels     map[string]*amqp.Channel
	channelMutex sync.Mutex
}

// Config holds the RabbitMQ connection parameters
type Config struct {
	Host     string
	User     string
	Password string
	Port     string
	VHost    string
}

var (
	// singleClient is the shared RabbitMQ client instance
	singleClient *RabbitMQClient

	// clientMutex protects singleClient initialization
	clientMutex sync.Mutex

	// initialized indicates whether the client has been initialized
	initialized bool
)

// NewConfig creates a Config from environment variables
func NewConfig() Config {
	return Config{
		Host:     getEnvOrDefault("rabbitmq_host", "localhost"),
		User:     getEnvOrDefault("rabbitmq_user", "guest"),
		Password: getEnvOrDefault("rabbitmq_pass", "guest"),
		Port:     getEnvOrDefault("rabbitmq_port", "5672"),
		VHost:    getEnvOrDefault("rabbitmq_vhost", ""),
	}
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// InitializeClient creates the shared RabbitMQ client instance
// This should be called during application startup
func InitializeClient(ctx context.Context) (*RabbitMQClient, error) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	log.Println("🔄 Init rmq client")

	if initialized && singleClient != nil {
		log.Println("✅ Already up rmq client")
		return singleClient, nil
	}

	config := NewConfig()

	log.Println("📦 Config created for new rmq client")
	log.Printf("📡 RabbitMQ URI: amqp://%s:***@%s:%s%s\n", config.User, config.Host, config.Port, config.VHost)

	client, err := InitializeClientWithConfig(ctx, config)
	if err != nil {
		log.Printf("❌ Failed to initialize RMQ client: %v\n", err)
		return nil, err
	}

	// This is redundant if InitializeClientWithConfig sets these values, but it's safer to have it here too
	singleClient = client
	initialized = true

	log.Println("✅ RabbitMq Client Connected || Configured")
	return client, nil
}

// InitializeClientWithConfig creates the shared RabbitMQ client with custom config
func InitializeClientWithConfig(ctx context.Context, config Config) (*RabbitMQClient, error) {
	amqpURI := fmt.Sprintf("amqp://%s:%s@%s:%d%s", config.User, config.Password, config.Host, config.Port, config.VHost)
	logger.WithCtx(ctx).Printf("🔗 Dialing RabbitMQ at %s\n", strings.Replace(amqpURI, config.Password, "***", 1))

	client, err := NewRabbitMQClient(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize RabbitMQ client: %v", err)
	}

	// Set the global client and mark as initialized
	singleClient = client
	initialized = true

	return client, nil
}

func NewRabbitMQClient(config Config) (*RabbitMQClient, error) {

	connStr := fmt.Sprintf("amqp://%s:%s@%s:%s/%s",
		config.User, config.Password, config.Host, config.Port, config.VHost)

	conn, err := amqp.Dial(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	client := &RabbitMQClient{
		conn:      conn,
		err:       make(chan error),
		channels:  make(map[string]*amqp.Channel),
		connected: true,
	}

	// Add both publisher and consumer channels
	if err := client.AddChannel("publisher"); err != nil {
		return nil, fmt.Errorf("failed to create publisher channel: %w", err)
	}

	if err := client.AddChannel("consumer"); err != nil {
		return nil, fmt.Errorf("failed to create consumer channel: %w", err)
	}

	return client, nil
}

func (c *RabbitMQClient) AddChannel(name string) error {
	c.channelMutex.Lock()
	defer c.channelMutex.Unlock()

	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}

	c.channels[name] = ch
	return nil
}

// GetClient returns the shared RabbitMQ client instance
// If the client hasn't been initialized, it returns nil
func GetClient() *RabbitMQClient {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if !initialized || singleClient == nil {
		log.Printf("GetClient: initialized=%v, singleClient is nil=%v",
			initialized, singleClient == nil)
		return nil
	}
	return singleClient
}

// CloseClient gracefully shuts down the shared RabbitMQ client
func CloseClient() error {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if initialized && singleClient != nil {
		err := singleClient.Close()
		singleClient = nil
		initialized = false
		return err
	}

	return nil
}

// NewClient creates a new RabbitMQ client with automatic reconnection
func NewClient() (*RabbitMQClient, error) {
	config := NewConfig()
	return NewClientWithConfig(config)
}

// NewClientWithConfig creates a new RabbitMQ client with the specified config
func NewClientWithConfig(config Config) (*RabbitMQClient, error) {
	client := &RabbitMQClient{
		err:        make(chan error),
		maxRetries: 5,
		channels:   make(map[string]*amqp.Channel),
	}

	err := client.Connect(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ client: %w", err)
	}

	return client, nil
}

// Connect establishes a connection to RabbitMQ
func (c *RabbitMQClient) Connect(config Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	amqpURI := fmt.Sprintf("amqp://%s:%s@%s:%s/%s",
		config.User, config.Password, config.Host, config.Port, config.VHost)

	conn, err := amqp.Dial(amqpURI)

	if err != nil {
		return fmt.Errorf("error connecting to RabbitMQ with %s: %w", amqpURI, err)
	}

	c.conn = conn
	c.connected = true
	c.reconnects = 0

	// Monitor connection for closure
	go func() {
		<-c.conn.NotifyClose(make(chan *amqp.Error))
		c.connected = false
		c.err <- errors.New("connection closed")
	}()

	return nil
}

// Reconnect attempts to reconnect with exponential backoff
func (c *RabbitMQClient) Reconnect() error {
	if c.reconnects >= c.maxRetries {
		return fmt.Errorf("max reconnection attempts (%d) reached", c.maxRetries)
	}

	config := NewConfig()

	// Exponential backoff for reconnection attempts
	backoff := time.Duration(1<<uint(c.reconnects)) * time.Second
	log.Printf("Attempting to reconnect to RabbitMQ in %v (attempt %d/%d)",
		backoff, c.reconnects+1, c.maxRetries)

	time.Sleep(backoff)
	c.reconnects++

	return c.Connect(config)
}

// Publish sends a message to the specified exchange/queue
func (c *RabbitMQClient) Publish(ctx context.Context, queueName string, payload interface{}, priority uint8) error {
	// Check for connection errors and reconnect if necessary
	select {
	case err := <-c.err:
		if err != nil {
			if err := c.Reconnect(); err != nil {
				return fmt.Errorf("failed to reconnect to RabbitMQ: %w", err)
			}
		}
	default:
		// Connection is ok, continue
	}

	if !c.connected {
		if err := c.Reconnect(); err != nil {
			return fmt.Errorf("failed to reconnect to RabbitMQ: %w", err)
		}
	}

	// Get a channel
	c.mu.Lock()
	ch, err := c.conn.Channel()
	c.mu.Unlock()

	if err != nil {
		return fmt.Errorf("error opening RabbitMQ channel: %w", err)
	}
	defer ch.Close()

	// Setup exchange and queue
	exchange := queueName
	routingKey := queueName
	exchangeType := "direct"

	err = ch.ExchangeDeclare(
		exchange,     // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange %s: %w", exchange, err)
	}

	// Create the queue
	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	// Bind queue to exchange
	err = ch.QueueBind(
		q.Name,     // queue name
		routingKey, // routing key
		exchange,   // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue %s to exchange %s: %w", queueName, exchange, err)
	}

	// Marshal the payload
	message, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling payload to JSON: %w", err)
	}

	// Publish the message
	err = ch.PublishWithContext(
		ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         message,
			Priority:     priority,
		})

	if err != nil {
		return fmt.Errorf("error publishing message: %w", err)
	}

	return nil
}

// Consume creates a consumer for the specified queue
func (c *RabbitMQClient) Consume(ctx context.Context, queueName string, consumerName string) (<-chan amqp.Delivery, error) {
	// Check for connection errors and reconnect if necessary
	select {
	case err := <-c.err:
		if err != nil {
			if err := c.Reconnect(); err != nil {
				return nil, fmt.Errorf("failed to reconnect to RabbitMQ: %w", err)
			}
		}
	default:
		// Connection is ok, continue
	}

	if !c.connected {
		if err := c.Reconnect(); err != nil {
			return nil, fmt.Errorf("failed to reconnect to RabbitMQ: %w", err)
		}
	}

	// Get a channel
	c.mu.Lock()
	ch, err := c.conn.Channel()
	c.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("error opening RabbitMQ channel: %w", err)
	}

	// Setup exchange and queue
	exchange := queueName
	routingKey := queueName
	exchangeType := "direct"

	err = ch.ExchangeDeclare(
		exchange,     // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange %s: %w", exchange, err)
	}

	// Create the queue
	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	// Bind queue to exchange
	err = ch.QueueBind(
		q.Name,     // queue name
		routingKey, // routing key
		exchange,   // exchange
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind queue %s to exchange %s: %w", queueName, exchange, err)
	}

	// Set prefetch count (QoS)
	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	// Start consuming
	deliveries, err := ch.Consume(
		q.Name,       // queue
		consumerName, // consumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register consumer: %w", err)
	}

	// We need to handle closing the channel when the context is done
	go func() {
		<-ctx.Done()
		ch.Close()
	}()

	return deliveries, nil
}

// Close closes the RabbitMQ connection
func (c *RabbitMQClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Connection returns the underlying amqp.Connection
func (c *RabbitMQClient) Connection() *amqp.Connection {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn
}
