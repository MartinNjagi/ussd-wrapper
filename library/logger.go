package library

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
)

var log = logrus.New()

func init() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local" // default fallback
	}

	log.SetFormatter(&logrus.JSONFormatter{})

	log.SetOutput(&lumberjack.Logger{
		Filename:   fmt.Sprintf("/var/log/myapp/app.%s.log", env),
		MaxSize:    10, // megabytes
		MaxBackups: 50,
		MaxAge:     28,   // days
		Compress:   true, // gzip
	})
}

func LogWithTrace(ctx context.Context) *logrus.Entry {
	return log.WithField("trace_id", GetTraceID(ctx))
}

func GetTraceID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	return spanCtx.TraceID().String()
}
