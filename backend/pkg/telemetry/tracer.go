// Package telemetry provides OpenTelemetry tracing configuration
package telemetry

import (
	"context"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerConfig holds configuration for OpenTelemetry tracer
type TracerConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Enabled        bool
}

// ShutdownFunc is a function that shuts down the tracer provider
type ShutdownFunc func(context.Context) error

// InitTracer initializes the OpenTelemetry tracer provider
// It reads OTEL_EXPORTER_OTLP_ENDPOINT and OTEL_EXPORTER_OTLP_HEADERS from environment
func InitTracer(cfg TracerConfig) (ShutdownFunc, error) {
	if !cfg.Enabled {
		log.Println("OpenTelemetry tracing is disabled")
		return func(ctx context.Context) error { return nil }, nil
	}

	// Check if OTLP endpoint is configured
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		log.Println("OTEL_EXPORTER_OTLP_ENDPOINT not set, tracing disabled")
		return func(ctx context.Context) error { return nil }, nil
	}

	ctx := context.Background()

	// Create OTLP HTTP exporter
	// The exporter automatically reads these environment variables:
	// - OTEL_EXPORTER_OTLP_ENDPOINT
	// - OTEL_EXPORTER_OTLP_HEADERS
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create tracer provider with batching
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set as global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Printf("OpenTelemetry tracing initialized: endpoint=%s, service=%s", endpoint, cfg.ServiceName)

	// Return shutdown function
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return tp.Shutdown(ctx)
	}, nil
}

// Tracer returns a tracer for the given component
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
