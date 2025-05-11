package library

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

var log = logrus.New()

func init() {
	log.SetFormatter(&logrus.JSONFormatter{})
}

func LogWithTrace(ctx context.Context) *logrus.Entry {
	return logrus.WithField("trace_id", GetTraceID(ctx))
}

func GetTraceID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	return spanCtx.TraceID().String()
}

// Middleware for injecting context-aware logging
func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			log.WithFields(logrus.Fields{
				"method": req.Method,
				"uri":    req.RequestURI,
			}).Info("Incoming request")
			return next(c)
		}
	}
}
