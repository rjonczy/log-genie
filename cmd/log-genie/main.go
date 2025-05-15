package loggenie

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rjonczy/log-genie/pkg/logger"
)

const (
	defaultRate              = 10
	defaultVerbosity         = "info"
	defaultTelemetryEndpoint = "collector:4318"
)

// Main is the entry point for the application
func Main() {
	// Parse command line flags
	rate := flag.Int("rate", defaultRate, "Number of logs per second")
	verbosity := flag.String("verbosity", defaultVerbosity, "Log verbosity level: debug, info, warn, error")
	telemetryEnabled := flag.Bool("telemetry", false, "Enable OpenTelemetry logs export")
	telemetryEndpoint := flag.String("telemetry-endpoint", defaultTelemetryEndpoint, "OpenTelemetry collector endpoint")
	localLogs := flag.Bool("local-logs", false, "Enable local logs to stdout/stderr even when telemetry is enabled")
	showResponses := flag.Bool("show-responses", false, "Show responses from the OTEL collector")
	flag.Parse()

	// Check environment variables (override command line flags if present)
	if envRate := os.Getenv("LOG_GENIE_RATE"); envRate != "" {
		if r, err := strconv.Atoi(envRate); err == nil {
			*rate = r
		}
	}

	if envVerbosity := os.Getenv("LOG_GENIE_VERBOSITY"); envVerbosity != "" {
		*verbosity = envVerbosity
	}

	if envTelemetry := os.Getenv("LOG_GENIE_TELEMETRY"); envTelemetry != "" {
		*telemetryEnabled = strings.ToLower(envTelemetry) == "true" || envTelemetry == "1"
	}

	if envTelemetryEndpoint := os.Getenv("LOG_GENIE_TELEMETRY_ENDPOINT"); envTelemetryEndpoint != "" {
		*telemetryEndpoint = envTelemetryEndpoint
	}

	if envLocalLogs := os.Getenv("LOG_GENIE_LOCAL_LOGS"); envLocalLogs != "" {
		*localLogs = strings.ToLower(envLocalLogs) == "true" || envLocalLogs == "1"
	}

	if envShowResponses := os.Getenv("LOG_GENIE_SHOW_RESPONSES"); envShowResponses != "" {
		*showResponses = strings.ToLower(envShowResponses) == "true" || envShowResponses == "1"
	}

	// Create logger
	config := logger.Config{
		Verbosity:         *verbosity,
		Rate:              *rate,
		TelemetryEnabled:  *telemetryEnabled,
		TelemetryEndpoint: *telemetryEndpoint,
		LocalLogEnabled:   *localLogs,
		ShowResponses:     *showResponses,
	}

	log, err := logger.New(config)
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		// Continue with local logging
	}
	defer log.Shutdown()

	// Calculate the interval between log emissions
	interval := time.Second / time.Duration(*rate)

	// Setup ticker for regular logs
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Setup signal catching
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Log startup message
	telemetryStatus := "disabled"
	if *telemetryEnabled {
		telemetryStatus = "enabled, endpoint: " + *telemetryEndpoint
	}
	localLogsStatus := "enabled"
	if !*localLogs {
		localLogsStatus = "disabled"
	}
	showResponsesStatus := "disabled"
	if *showResponses {
		showResponsesStatus = "enabled"
	}

	startupLog := log.WithField("app", "log-genie")
	startupLog.Info(fmt.Sprintf("Starting log generation at %d logs per second with %s verbosity. OpenTelemetry: %s. Local logs: %s. Show responses: %s",
		*rate, *verbosity, telemetryStatus, localLogsStatus, showResponsesStatus))

	// Run the log generator
	go func() {
		for range ticker.C {
			// Occasionally generate an error log (about 5% of the time)
			if time.Now().UnixNano()%20 == 0 {
				log.GenerateRandomErrorLog()
			} else {
				log.GenerateRandomLog()
			}
		}
	}()

	// Wait for termination signal
	<-sigs
	fmt.Println("Shutting down log generator")
}
