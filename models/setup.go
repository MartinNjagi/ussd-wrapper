package models

import (
	"database/sql"
	"github.com/go-redis/redis/v8"
	amqp "github.com/rabbitmq/amqp091-go"
	"sync"
)

type AppContext struct {
	DB         *sql.DB
	DBSlave    *sql.DB
	Redis      *redis.Client
	RabbitConn *amqp.Connection
	RabbitChan *amqp.Channel
}

// RabbitMQConn is the connection created
type RabbitMQConn struct {
	conn *amqp.Connection
	err  chan error
}

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
