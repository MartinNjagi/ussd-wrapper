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
func (ctl *Controller) RegisterRoutes1(e *echo.Echo) {
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

// controller/common.go

// RegisterRoutes registers all application routes
func (ctl *Controller) RegisterRoutes(e *echo.Echo) {
	// Main USSD callback handler - all USSD interactions go through here
	e.POST("/ussd/callback", ctl.HandleUSSD)

	// API routes with authentication middleware
	//api := e.Group("/api", middleware.JWT([]byte(ctl.config.JWTSecret)))

	// Admin routes
	/*admin := api.Group("/admin")
	admin.GET("/users", ctl.GetUsers)
	admin.GET("/users/:id", ctl.GetUser)
	admin.POST("/users", ctl.CreateUser)
	admin.PUT("/users/:id", ctl.UpdateUser)
	admin.DELETE("/users/:id", ctl.DeleteUser)*/

	// Transaction management
	/*admin.GET("/transactions", ctl.ListTransactions)
	admin.GET("/transactions/:id", ctl.GetTransaction)
	admin.PUT("/transactions/:id/status", ctl.UpdateTransactionStatus)*/

	// System configuration
	/*admin.GET("/config", ctl.GetSystemConfig)
	admin.PUT("/config", ctl.UpdateSystemConfig)
	admin.GET("/ussd-menus", ctl.GetUSSDMenus)
	admin.PUT("/ussd-menus", ctl.UpdateUSSDMenus)*/

	// Reporting
	/*reports := api.Group("/reports")
	reports.GET("/transactions", ctl.TransactionReport)
	reports.GET("/users", ctl.UserReport)
	reports.GET("/usage", ctl.UsageReport)
	reports.GET("/revenue", ctl.RevenueReport)
	reports.GET("/audit-logs", ctl.AuditLogReport)*/

	// Webhooks for external service callbacks
	/*hooks := e.Group("/webhooks")
	hooks.POST("/payment-notification", ctl.PaymentNotification)
	hooks.POST("/sms-delivery", ctl.SMSDeliveryStatus)*/

	// Health check
	//e.GET("/health", ctl.HealthCheck)
}
