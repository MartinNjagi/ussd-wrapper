package library

import (
	"context"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

func InitTracer() (context.Context, error) {
	ctx := context.Background()

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("otel-collector:4318"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return ctx, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("ussd-wrapper"),
		),
	)
	if err != nil {
		return ctx, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	return ctx, nil
}

// TraceMiddleware injects a span into each request context
func TraceMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tracer := otel.Tracer("ussd-wrapper")

		req := c.Request()
		ctx := req.Context()

		// Start a new span for the request
		ctx, span := tracer.Start(ctx, req.URL.Path)
		defer span.End()

		// Add metadata to span
		span.SetAttributes(
			attribute.String("http.method", req.Method),
			attribute.String("http.path", req.URL.Path),
			attribute.String("request_id", c.Response().Header().Get(echo.HeaderXRequestID)),
		)

		// Inject context into request for downstream usage
		c.SetRequest(req.WithContext(ctx))
		return next(c)
	}
}

// SetupTracer initializes OpenTelemetry tracing
func SetupTracer() trace.Tracer {
	return otel.Tracer("ussd-wrapper")
}

type AmqpHeadersCarrier map[string]interface{}

func (a AmqpHeadersCarrier) Get(key string) string {
	v, ok := a[key]
	if !ok {
		return ""
	}
	return v.(string)
}

func (a AmqpHeadersCarrier) Set(key string, value string) {
	a[key] = value
}

func (a AmqpHeadersCarrier) Keys() []string {
	i := 0
	r := make([]string, len(a))

	for k := range a {
		r[i] = k
		i++
	}

	return r
}

// InjectAMQPHeaders injects the tracing from the context into the header map
func InjectAMQPHeaders(ctx context.Context) map[string]interface{} {
	h := make(AmqpHeadersCarrier)
	otel.GetTextMapPropagator().Inject(ctx, h)
	return h
}

// ExtractAMQPHeaders extracts the tracing from the header and puts it into the context
func ExtractAMQPHeaders(ctx context.Context, headers map[string]interface{}) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, AmqpHeadersCarrier(headers))
}
