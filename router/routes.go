package router

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	echoSwagger "github.com/swaggo/echo-swagger"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"ussd-wrapper/connections"
	"ussd-wrapper/controller"
	"ussd-wrapper/library"
	"ussd-wrapper/queue"
)

func Init() error {
	// 🟣 1. Tracer Setup
	ctx, err := library.InitTracer()
	if err != nil {
		return err
	}
	tracer := library.SetupTracer()

	// Create a root span (a trace) to measure some operation.
	ctx, main := tracer.Start(ctx, "ussd-wrapper")
	// End the span when the operation we are measuring is done.
	defer main.End()

	// 🟡 2. Database Connections and Migrations
	dbInstance := connections.DbInstance()
	dbSlave := connections.DbInstanceSlave()

	driver, err := mysql.WithInstance(dbInstance, &mysql.Config{})
	if err != nil {

		logrus.Panic(err)
	}

	m, err := migrate.NewWithDatabaseInstance(fmt.Sprintf("file:///%s/migrations", GetRootPath()), "mysql", driver)
	if err != nil {

		logrus.Errorf("migration setup error %s ", err.Error())
	}

	err = m.Up() // or m.Step(2) if you want to explicitly set the number of migrations to run
	if err != nil {

		logrus.Errorf("migration error %s ", err.Error())
	}

	// 🟢 3. Redis
	redisClient := connections.InitRedis()

	// 🔵 4. RabbitMQ Connection
	rabbitConn, err := connections.InitializeClient()
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ: %v", err)
	}

	// Create the queue manager
	queueManager, err := queue.NewQueueManager(tracer, dbInstance, dbSlave, redisClient)
	if err != nil {
		log.Fatalf("Failed to create queue manager: %v", err)
	}

	// Start all consumers
	go queueManager.InitializeQueues(ctx)

	// 🔗 5. Create Controller with dependencies
	ctrl := controller.NewController(dbInstance, dbSlave, redisClient, rabbitConn, tracer)

	// 🚀 6. Echo Setup
	e := echo.New()
	e.Static("/doc", "api")
	e.Use(middleware.Gzip())
	e.IPExtractor = echo.ExtractIPFromXFFHeader()
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(library.TraceMiddleware)

	// setup log format and parameters to log for every request
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {

			req := c.Request()
			res := c.Response()
			start := values.StartTime
			startMicro := start.UnixMicro()

			stop := time.Now()
			stopMicro := stop.UnixMicro()

			id := req.Header.Get(echo.HeaderXRequestID)
			if id == "" {

				id = res.Header().Get(echo.HeaderXRequestID)
			}

			reqSize := req.Header.Get(echo.HeaderContentLength)
			if reqSize == "" {

				reqSize = "0"
			}

			traceID := req.Header.Get("trace-id")
			if traceID == "" {

				traceID = "0"
			}

			service, _ := os.Hostname()

			logrus.WithContext(c.Request().Context()).WithFields(logrus.Fields{
				"service":  service,
				"id":       id,
				"ip":       c.RealIP(),
				"time":     stop.Format(time.RFC3339),
				"host":     req.Host,
				"method":   req.Method,
				"uri":      req.RequestURI,
				"status":   res.Status,
				"size":     reqSize,
				"referer":  req.Referer(),
				"ua":       req.UserAgent(),
				"ttl":      stopMicro - startMicro,
				"trace-id": traceID,
			}).Info("API Response")

			return nil
		},
	}))

	allowedMethods := []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete, http.MethodOptions}
	AllowOrigins := []string{"*"}

	//setup CORS
	corsConfig := middleware.CORSConfig{
		AllowOrigins: AllowOrigins, // in production limit this to only known hosts
		AllowHeaders: AllowOrigins,
		AllowMethods: allowedMethods,
	}

	e.Use(middleware.CORSWithConfig(corsConfig))
	// 📘 7. Swagger UI
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// 🧩 8. Register Routes
	registerRoutes(e, ctrl)

	// 🔄 9. Start Echo server
	log.Println("✅ USSD Wrapper Server running on :8080")
	return e.Start(":8080")
}

func GetRootPath() string {

	_, b, _, _ := runtime.Caller(0)

	// Root folder of this project
	return filepath.Join(filepath.Dir(b), "./")
}

// Now we'll use the controller's RegisterRoutes method
func registerRoutes(e *echo.Echo, ctrl *controller.Controller) {
	ctrl.RegisterRoutes(e)
}
