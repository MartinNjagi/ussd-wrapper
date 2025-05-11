// controller/common.go
package controller

import (
	"database/sql"
	"github.com/go-redis/redis"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/trace"
	"ussd-wrapper/connections"
)

// Controller holds all dependencies needed by controller handlers
type Controller struct {
	db         *sql.DB
	dbSlave    *sql.DB
	redis      *redis.Client
	rabbitConn *connections.RabbitMQClient
	tracer     trace.Tracer
}

// NewController creates a new controller with all required dependencies
func NewController(db *sql.DB, dbSlave *sql.DB, redis *redis.Client, rabbitConn *connections.RabbitMQClient, tracer trace.Tracer) *Controller {
	return &Controller{
		db:         db,
		dbSlave:    dbSlave,
		redis:      redis,
		tracer:     tracer,
		rabbitConn: rabbitConn,
	}
}

// RegisterRoutes registers all application routes
func (ctl *Controller) RegisterRoutes(e *echo.Echo) {
	e.POST("/ussd/callback", ctl.HandleUSSD)
	// Add more routes as needed, grouped by functionality:

	// Example: Auth routes
	// auth := e.Group("/auth")
	// auth.POST("/login", ctl.Login)
	// auth.POST("/logout", ctl.Logout)

	// Example: API routes
	// api := e.Group("/api")
	// api.GET("/users", ctl.GetUsers)
}
