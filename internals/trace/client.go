package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func SetupTracer() (*sdktrace.TracerProvider, error, func()) {
	exporter, err := otlptracehttp.New(context.Background())
	if err != nil {
		return nil, err, func() {
			// No-op shutdown function if exporter creation fails
		}
	}
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("go-chi-server"),
	)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
	)

	// tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
	otel.SetTracerProvider(tp)
	return tp, nil, func() {
		_ = tp.Shutdown(context.Background())
	}
}

func NewTracer(name string) {
	context := context.Background()
	tracer := otel.Tracer(name)
	context, span := tracer.Start(context, "HandleClientRequest")
	defer span.End()
}
