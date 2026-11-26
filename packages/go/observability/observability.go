// Package observability configures OpenTelemetry providers for services.
//
// Start configures tracing, metrics, and logs for process.
// It does not instrument HTTP routes automatically.
// To create inbound HTTP spans, wrap router with otelhttp handler.
//
// Example:
//
//	shutdown, err := observability.Start(ctx, observability.Config{})
//	if err != nil {
//		return fmt.Errorf("setup observability: %w", err)
//	}
//	defer func() {
//		_ = shutdown(context.Background())
//	}()
//	server := &http.Server{Handler: otelhttp.NewHandler(router, "/")}
package observability

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

type Config struct {
	MetricsExportInterval time.Duration
}

func Start(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if strings.EqualFold(os.Getenv("OTEL_SDK_DISABLED"), "true") {
		return func(context.Context) error { return nil }, nil
	}

	var shutdownFuncs []func(context.Context) error
	var err error

	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	otel.SetTextMapPropagator(newPropagator())

	tracerProvider, err := newTracerProvider(ctx)
	if err != nil {
		handleErr(err)
		return shutdown, err
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	meterProvider, err := newMeterProvider(ctx, cfg)
	if err != nil {
		handleErr(err)
		return shutdown, err
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	loggerProvider, err := newLoggerProvider(ctx)
	if err != nil {
		handleErr(err)
		return shutdown, err
	}
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	otel.SetLoggerProvider(loggerProvider)

	return shutdown, err
}

func NewHTTPHandler(handler http.Handler, operation string) http.Handler {
	return otelhttp.NewHandler(handler, operation)
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newResource(ctx context.Context) (*resource.Resource, error) {
	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
	)
	if err != nil {
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}

	return res, nil
}

func newTracerProvider(ctx context.Context) (*trace.TracerProvider, error) {
	res, err := newResource(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize tracer resource: %w", err)
	}

	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	return trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(exp),
	), nil
}

func newMeterProvider(ctx context.Context, cfg Config) (*metric.MeterProvider, error) {
	res, err := newResource(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize meter resource: %w", err)
	}

	exp, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create metric exporter: %w", err)
	}

	interval := cfg.MetricsExportInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}

	return metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(
			exp,
			metric.WithInterval(interval),
		)),
	), nil
}

func newLoggerProvider(ctx context.Context) (*sdklog.LoggerProvider, error) {
	res, err := newResource(ctx)
	if err != nil {
		return nil, fmt.Errorf("create log resource: %w", err)
	}

	exp, err := otlploghttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create log exporter: %w", err)
	}

	return sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(
			sdklog.NewBatchProcessor(exp),
		),
	), nil
}
