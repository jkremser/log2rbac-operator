package internal

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.6.1"
	"strconv"
)

func SetupTracing(cfg Config, ctx context.Context, log logr.Logger) func() {
	if !cfg.Tracing.Enabled {
		log.Info("OTLP tracing is disabled")
		return func() {}
	}
	log.Info("OTLP tracing is ON")
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(cfg.Tracing.Endpoint),
		otlptracehttp.WithInsecure(),
		)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		log.Error(err, "creating OTLP trace exporter")
	}
	var samplerOption sdktrace.TracerProviderOption
	if r, err := strconv.ParseFloat(cfg.Tracing.SamplingRatio, 64); err == nil {
		samplerOption = sdktrace.WithSampler(sdktrace.TraceIDRatioBased(r))
		log.Info(fmt.Sprintf( "Tracing: sampling ratio is set to '%f'", r))
	} else {
		log.Info( "Tracing: sampling ratio is not specified, using AlwaysSample")
		samplerOption = sdktrace.WithSampler(sdktrace.AlwaysSample())
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(newResource(cfg.App.Version)),
		samplerOption,
	)
	otel.SetTracerProvider(tracerProvider)

	return func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			log.Error(err,  "stopping tracer provider")
		}
	}
}

func newResource(version string) *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("log2rbac"),
		semconv.ServiceVersionKey.String(version),
	)
}
