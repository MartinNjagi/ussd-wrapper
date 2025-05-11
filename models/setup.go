package models

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQConn is the connection created
type RabbitMQConn struct {
	conn *amqp.Connection
	err  chan error
}
