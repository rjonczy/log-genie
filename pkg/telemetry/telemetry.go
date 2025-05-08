package telemetry

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// Provider is a wrapper for OpenTelemetry log provider
type Provider struct {
	enabled     bool
	endpoint    string
	logProvider *sdklog.LoggerProvider
	logger      log.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	logCount    atomic.Int64
	mutex       sync.Mutex
	lastReport  time.Time
}

// Config holds the configuration for the telemetry provider
type Config struct {
	Enabled  bool
	Endpoint string
}

// LogLevel represents the level of logging
type LogLevel string

const (
	// DebugLevel is the debug level
	DebugLevel LogLevel = "debug"
	// InfoLevel is the info level
	InfoLevel LogLevel = "info"
	// WarnLevel is the warn level
	WarnLevel LogLevel = "warn"
	// ErrorLevel is the error level
	ErrorLevel LogLevel = "error"
)

// New creates a new telemetry provider
func New(config Config) (*Provider, error) {
	p := &Provider{
		enabled:    config.Enabled,
		endpoint:   config.Endpoint,
		lastReport: time.Now(),
	}

	if !p.enabled {
		return p, nil
	}

	var err error
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// Configure OTLP HTTP exporter
	var exporter sdklog.Exporter

	// Parse the endpoint URL
	insecure := true // Default to insecure for easier testing
	options := []otlploghttp.Option{
		otlploghttp.WithEndpoint(p.endpoint),
	}

	if insecure {
		options = append(options, otlploghttp.WithInsecure())
	}

	exporter, err = otlploghttp.New(p.ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create batch processor with exporter
	batchProcessor := sdklog.NewBatchProcessor(
		exporter,
		// Configure batch settings
		sdklog.WithExportTimeout(5*time.Second),
		sdklog.WithMaxQueueSize(2048),
		sdklog.WithExportMaxBatchSize(512),
	)

	// Create log provider with BatchProcessor
	p.logProvider = sdklog.NewLoggerProvider(
		sdklog.WithProcessor(batchProcessor),
	)

	// Get a logger instance
	p.logger = p.logProvider.Logger("log-genie")

	// Reset the log counter
	p.logCount.Store(0)

	// Report logs sent every minute
	go p.reportLogsSent()

	return p, nil
}

// reportLogsSent reports the number of logs sent periodically
func (p *Provider) reportLogsSent() {
	if !p.enabled {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			count := p.logCount.Load()
			now := time.Now()
			elapsed := now.Sub(p.lastReport).Seconds()
			if elapsed > 0 {
				rate := float64(count) / elapsed
				fmt.Printf("TELEMETRY: Sent %d logs in the last %.1f seconds (%.1f logs/sec)\n",
					count, elapsed, rate)

				// Reset counter and update last report time
				p.logCount.Store(0)
				p.lastReport = now
			}
		case <-p.ctx.Done():
			return
		}
	}
}

// Shutdown shuts down the telemetry provider
func (p *Provider) Shutdown() {
	if p.cancel != nil {
		p.cancel()
	}

	if p.logProvider != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = p.logProvider.Shutdown(ctx)
	}
}

// SendLog sends a log to the telemetry provider
func (p *Provider) SendLog(level LogLevel, message string, fields map[string]interface{}) error {
	if !p.enabled || p.logger == nil {
		return fmt.Errorf("telemetry is not enabled or logger is not initialized")
	}

	// Create a new record
	record := &log.Record{}

	// Set the timestamp
	record.SetTimestamp(time.Now())

	// Set severity based on log level
	var severity log.Severity
	switch level {
	case DebugLevel:
		severity = log.SeverityDebug
	case InfoLevel:
		severity = log.SeverityInfo
	case WarnLevel:
		severity = log.SeverityWarn
	case ErrorLevel:
		severity = log.SeverityError
	}
	record.SetSeverity(severity)
	record.SetSeverityText(string(level))

	// Set the message body
	record.SetBody(log.StringValue(message))

	// Add attributes from fields
	attributes := make([]log.KeyValue, 0, len(fields))
	for k, v := range fields {
		switch val := v.(type) {
		case string:
			attributes = append(attributes, log.String(k, val))
		case int:
			attributes = append(attributes, log.Int(k, val))
		case int64:
			attributes = append(attributes, log.Int64(k, val))
		case float64:
			attributes = append(attributes, log.Float64(k, val))
		case bool:
			attributes = append(attributes, log.Bool(k, val))
		default:
			attributes = append(attributes, log.String(k, fmt.Sprintf("%v", val)))
		}
	}
	record.AddAttributes(attributes...)

	// Emit the log record
	p.logger.Emit(p.ctx, *record)

	// Increment counter
	p.logCount.Add(1)

	return nil
}

// IsEnabled returns whether telemetry is enabled
func (p *Provider) IsEnabled() bool {
	return p.enabled
}

// GetLogCount returns the current log count
func (p *Provider) GetLogCount() int64 {
	return p.logCount.Load()
}
