// Package telemetry configures OpenTelemetry tracing and metrics.
package telemetry

import (
	"context"
	"log/slog"

	"github.com/forgebox/forgebox/internal/config"
)

var shutdownFn func()

// Init sets up OpenTelemetry tracing and metrics exporters.
func Init(cfg config.TelemetryConfig) error {
	if cfg.OTLPEndpoint == "" {
		slog.Debug("telemetry disabled (no OTLP endpoint)")
		return nil
	}

	// TODO: Initialize OpenTelemetry SDK:
	// - TracerProvider with OTLP gRPC exporter
	// - MeterProvider with OTLP gRPC exporter or Prometheus
	// - Set global providers
	//
	// Example:
	//   exporter, _ := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint))
	//   tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
	//   otel.SetTracerProvider(tp)
	//   shutdownFn = func() { tp.Shutdown(context.Background()) }

	slog.Info("telemetry initialized", "endpoint", cfg.OTLPEndpoint)
	return nil
}

// Shutdown flushes pending telemetry data and closes exporters.
func Shutdown() {
	if shutdownFn != nil {
		shutdownFn()
	}
}

// TraceVMBoot records a span for VM boot time.
func TraceVMBoot(ctx context.Context, vmID string) (context.Context, func()) {
	// TODO: Start a span using otel.Tracer("forgebox.vm").Start(ctx, "vm.boot")
	return ctx, func() {}
}

// TraceToolExec records a span for tool execution.
func TraceToolExec(ctx context.Context, toolName string) (context.Context, func()) {
	// TODO: Start a span using otel.Tracer("forgebox.tool").Start(ctx, "tool.exec")
	return ctx, func() {}
}

// TraceProviderCall records a span for an LLM provider call.
func TraceProviderCall(ctx context.Context, provider, model string) (context.Context, func()) {
	// TODO: Start a span using otel.Tracer("forgebox.provider").Start(ctx, "provider.call")
	return ctx, func() {}
}
