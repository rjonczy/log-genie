package telemetry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// Provider is a wrapper for OpenTelemetry log provider
type Provider struct {
	enabled       bool
	endpoint      string
	hostPort      string // Just the host:port part
	path          string // The path part
	logProvider   *sdklog.LoggerProvider
	logger        log.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	logCount      atomic.Int64
	mutex         sync.Mutex
	lastReport    time.Time
	httpClient    *http.Client
	showResponses bool // Flag to control response display
}

// Config holds the configuration for the telemetry provider
type Config struct {
	Enabled       bool
	Endpoint      string
	ShowResponses bool // New configuration field to control response display
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

// This section intentionally left empty after refactoring to use direct POST testing

// parseEndpoint separates host:port from path in an endpoint string
func parseEndpoint(endpoint string) (hostPort, path string) {
	// Handle case where the endpoint might already have a scheme
	if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.TrimPrefix(endpoint, "http://")
	} else if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.TrimPrefix(endpoint, "https://")
	}

	// Split at first slash to separate host:port from path
	parts := strings.SplitN(endpoint, "/", 2)
	hostPort = parts[0]

	if len(parts) > 1 {
		path = "/" + parts[1]
	} else {
		path = ""
	}

	return hostPort, path
}

// New creates a new telemetry provider
func New(config Config) (*Provider, error) {
	hostPort, path := parseEndpoint(config.Endpoint)

	p := &Provider{
		enabled:       config.Enabled,
		endpoint:      config.Endpoint,
		hostPort:      hostPort,
		path:          path,
		lastReport:    time.Time{}, // Initialize to zero time to trigger immediate status messages
		httpClient:    &http.Client{Timeout: 5 * time.Second},
		showResponses: config.ShowResponses,
	}

	if !p.enabled {
		return p, nil
	}

	var err error
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// If show responses is enabled, print configuration information
	if p.showResponses {
		fmt.Println("OTEL COLLECTOR CONFIG:")
		fmt.Printf("  - Endpoint: %s\n", p.endpoint)
		fmt.Printf("  - Host:Port: %s\n", p.hostPort)
		fmt.Printf("  - Path: %s\n", p.path)

		// Test direct POST to the collector
		go p.testDirectPost()
	}

	// Configure OTLP HTTP exporter
	var exporter sdklog.Exporter

	// For OTLP exporter, we need just the host:port part
	insecure := true // Default to insecure for easier testing
	options := []otlploghttp.Option{
		otlploghttp.WithEndpoint(p.hostPort),
	}

	// The WithHTTPClient option might not be available in this version
	// Instead, we'll rely on the custom transport to capture responses

	if insecure {
		options = append(options, otlploghttp.WithInsecure())
	}

	// If path was provided, add it to the URL path prefix
	if p.path != "" {
		options = append(options, otlploghttp.WithURLPath(p.path))
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
		// Use smaller batch size for more frequent POST operations
		sdklog.WithExportMaxBatchSize(10),
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

	// Set up periodic POST test if responses should be shown
	if p.showResponses {
		// Start a goroutine to periodically test direct POST
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					p.testDirectPost()
				case <-p.ctx.Done():
					return
				}
			}
		}()
	}

	return p, nil
}

// testDirectPost sends a test log directly to the collector using POST
// and displays the response
func (p *Provider) testDirectPost() {
	if !p.enabled || !p.showResponses {
		return
	}

	// First, test using curl-like direct POST
	pathToUse := "/v1/logs"
	if p.path != "" {
		pathToUse = p.path
	}
	logsUrl := fmt.Sprintf("http://%s%s", p.hostPort, pathToUse)

	fmt.Printf("DEBUG: Testing direct POST to %s\n", logsUrl)

	// Create a test log payload similar to what the OTLP exporter would send
	testPayload := `{
		"resourceLogs": [{
			"resource": {
				"attributes": [{
					"key": "service.name",
					"value": {"stringValue": "log-genie-test"}
				}]
			},
			"scopeLogs": [{
				"logRecords": [{
					"timeUnixNano": "1715777777000000000",
					"severityNumber": 9,
					"severityText": "INFO",
					"body": {"stringValue": "Test log message"},
					"attributes": [{
						"key": "test",
						"value": {"stringValue": "value"}
					}]
				}]
			}]
		}]
	}`

	// Remove whitespace to make it more compact
	testPayload = strings.ReplaceAll(testPayload, "\t", "")
	testPayload = strings.ReplaceAll(testPayload, "\n", "")

	// Create the POST request
	req, err := http.NewRequestWithContext(p.ctx, "POST", logsUrl,
		strings.NewReader(testPayload))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		resp, err := p.httpClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if len(body) > 0 {
				fmt.Printf("OTEL COLLECTOR DIRECT POST RESPONSE: %s\n", string(body))
			} else {
				fmt.Printf("DEBUG: OTLP collector returned empty response with status: %d\n",
					resp.StatusCode)
			}
		} else {
			fmt.Printf("DEBUG: Error sending test POST request: %v\n", err)
		}
	} else {
		fmt.Printf("DEBUG: Error creating test POST request: %v\n", err)
	}
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
