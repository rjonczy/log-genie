package logger

import (
	"os"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/rjonczy/log-genie/pkg/telemetry"
	"github.com/sirupsen/logrus"
)

// Logger is a wrapper around logrus.Logger
type Logger struct {
	*logrus.Logger
	telemetryEnabled bool
	telemetry        *telemetry.Provider
	localLogEnabled  bool
}

// Config holds the configuration for the logger
type Config struct {
	Verbosity         string
	Rate              int
	TelemetryEnabled  bool
	TelemetryEndpoint string
	LocalLogEnabled   bool
	ShowResponses     bool
	ApplicationID     string // Application ID for OTEL resource attributes
}

// LogLevel represents the level of logging
type LogLevel string

const (
	// Debug level
	Debug LogLevel = "debug"
	// Info level
	Info LogLevel = "info"
	// Warn level
	Warn LogLevel = "warn"
	// Error level
	Error LogLevel = "error"
)

// New creates a new logger with the given configuration
func New(config Config) (*Logger, error) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	// Set log level
	switch strings.ToLower(config.Verbosity) {
	case string(Debug):
		logger.SetLevel(logrus.DebugLevel)
	case string(Info):
		logger.SetLevel(logrus.InfoLevel)
	case string(Warn):
		logger.SetLevel(logrus.WarnLevel)
	case string(Error):
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	l := &Logger{
		Logger:           logger,
		telemetryEnabled: config.TelemetryEnabled,
		localLogEnabled:  config.LocalLogEnabled || !config.TelemetryEnabled, // If telemetry is disabled, local logs are always enabled
	}

	// Initialize telemetry provider if enabled
	if config.TelemetryEnabled {
		telemetryProvider, err := telemetry.New(telemetry.Config{
			Enabled:       true,
			Endpoint:      config.TelemetryEndpoint,
			ShowResponses: config.ShowResponses,
			ApplicationID: config.ApplicationID,
		})
		if err != nil {
			logger.WithError(err).Error("Failed to initialize telemetry provider, falling back to local logging")
			l.telemetryEnabled = false
			l.localLogEnabled = true
			return l, err
		}
		l.telemetry = telemetryProvider
		logger.Info("Telemetry provider initialized successfully")
	}

	return l, nil
}

// Shutdown gracefully shuts down the logger and its telemetry provider
func (l *Logger) Shutdown() {
	if l.telemetry != nil {
		l.telemetry.Shutdown()
	}
}

// GenerateRandomLog generates a random log entry
func (l *Logger) GenerateRandomLog() {
	// Generate a random log level
	levels := []LogLevel{Debug, Info, Warn, Error}
	level := levels[gofakeit.Number(0, len(levels)-1)]

	// Generate fake data
	message := gofakeit.Sentence(gofakeit.Number(5, 15))
	service := gofakeit.AppName()
	userID := gofakeit.UUID()
	httpMethod := gofakeit.HTTPMethod()
	statusCode := gofakeit.HTTPStatusCode()
	latency := gofakeit.Number(1, 500)
	ipAddress := gofakeit.IPv4Address()

	// Create log fields map
	fields := map[string]interface{}{
		"service":     service,
		"user_id":     userID,
		"http_method": httpMethod,
		"status_code": statusCode,
		"latency_ms":  latency,
		"ip_address":  ipAddress,
		"timestamp":   time.Now().UnixNano(),
	}

	// Send to telemetry if enabled
	if l.telemetryEnabled && l.telemetry != nil {
		var telemetryLevel telemetry.LogLevel
		switch level {
		case Debug:
			telemetryLevel = telemetry.DebugLevel
		case Info:
			telemetryLevel = telemetry.InfoLevel
		case Warn:
			telemetryLevel = telemetry.WarnLevel
		case Error:
			telemetryLevel = telemetry.ErrorLevel
		}

		err := l.telemetry.SendLog(telemetryLevel, message, fields)
		if err != nil {
			// If telemetry fails, log the error locally
			l.WithError(err).Error("Failed to send log to telemetry endpoint")
		}
	}

	// Log locally if enabled or if telemetry is not enabled
	if l.localLogEnabled {
		// Create log entry with random fields
		logEntry := l.WithFields(logrus.Fields(fields))

		// Log at the random level
		switch level {
		case Debug:
			logEntry.Debug(message)
		case Info:
			logEntry.Info(message)
		case Warn:
			logEntry.Warn(message)
		case Error:
			logEntry.Error(message)
		}
	}
}

// GenerateRandomErrorLog generates a random error log entry
func (l *Logger) GenerateRandomErrorLog() {
	// Generate fake data
	errorMessage := gofakeit.SentenceSimple()
	service := gofakeit.AppName()
	requestID := gofakeit.UUID()
	errorCode := gofakeit.Number(400, 599)
	stackTrace := gofakeit.LoremIpsumSentence(5)

	// Create fields map
	fields := map[string]interface{}{
		"service":     service,
		"request_id":  requestID,
		"error_code":  errorCode,
		"stack_trace": stackTrace,
		"timestamp":   time.Now().UnixNano(),
	}

	// Send to telemetry if enabled
	if l.telemetryEnabled && l.telemetry != nil {
		err := l.telemetry.SendLog(telemetry.ErrorLevel, errorMessage, fields)
		if err != nil {
			// If telemetry fails, log the error locally
			l.WithError(err).Error("Failed to send error log to telemetry endpoint")
		}
	}

	// Log locally if enabled or if telemetry is not enabled
	if l.localLogEnabled {
		// Create log entry with random fields
		logEntry := l.WithFields(logrus.Fields(fields))
		logEntry.Error(errorMessage)
	}
}

// WithField creates a new entry with the specified field
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}
