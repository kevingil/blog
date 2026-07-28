package telemetry

import (
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware creates a Fiber middleware that adds OpenTelemetry tracing
func TracingMiddleware(serviceName string) fiber.Handler {
	tracer := otel.Tracer(serviceName)

	return func(c *fiber.Ctx) error {
		// Extract context from incoming request headers
		ctx := otel.GetTextMapPropagator().Extract(c.Context(), propagation.HeaderCarrier(c.GetReqHeaders()))

		// Start a new span
		spanName := c.Method() + " " + c.Path()
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethod(c.Method()),
				semconv.HTTPTarget(c.OriginalURL()),
				semconv.HTTPRoute(c.Route().Path),
				semconv.NetHostName(c.Hostname()),
				attribute.String("http.user_agent", c.Get("User-Agent")),
			),
		)
		defer span.End()

		// Store span context in Fiber locals for downstream use
		c.Locals("otel-span", span)
		c.SetUserContext(ctx)

		// Execute the handler
		err := c.Next()

		// Record response status
		status := c.Response().StatusCode()
		span.SetAttributes(semconv.HTTPStatusCode(status))

		// Mark span as error if status >= 400
		if status >= 400 {
			span.SetAttributes(attribute.Bool("error", true))
		}

		// Record error if present
		if err != nil {
			span.RecordError(err)
		}

		return err
	}
}

// GetSpanFromContext retrieves the current span from Fiber context
func GetSpanFromContext(c *fiber.Ctx) trace.Span {
	if span, ok := c.Locals("otel-span").(trace.Span); ok {
		return span
	}
	return nil
}
