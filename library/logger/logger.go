package logger

import (
	"context"
	"go.opentelemetry.io/otel/trace"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var log = logrus.New()

const traceField = "trace_id"

func Setup(logDir string) error {
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetReportCaller(true)
	log.SetLevel(logrus.InfoLevel)

	// Output to console (stdout)
	log.SetOutput(os.Stdout)

	// Setup per-level rotating file hooks
	levels := []logrus.Level{
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.DebugLevel,
		logrus.FatalLevel,
	}

	for _, level := range levels {
		log.AddHook(NewLevelHook(level, filepath.Join(logDir, level.String()+".log")))
	}

	return nil
}

// NewLevelHook returns a logrus hook for a specific level with file rotation
func NewLevelHook(level logrus.Level, path string) logrus.Hook {
	writer := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    10, // megabytes
		MaxBackups: 50, // number of rotated files
		MaxAge:     28, // days
		Compress:   true,
	}

	return &LevelHook{
		Level:  level,
		Writer: writer,
	}
}

type LevelHook struct {
	Level  logrus.Level
	Writer io.Writer
}

func (h *LevelHook) Levels() []logrus.Level {
	return []logrus.Level{h.Level}
}

func (h *LevelHook) Fire(entry *logrus.Entry) error {
	line, err := entry.Logger.Formatter.Format(entry)
	if err != nil {
		return err
	}
	_, err = h.Writer.Write(line)
	return err
}

func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

func WithCtx(ctx context.Context) *logrus.Entry {
	tid := TraceID(ctx)
	if tid == "" {
		return log.WithFields(logrus.Fields{})
	}
	return log.WithField(traceField, tid)
}
