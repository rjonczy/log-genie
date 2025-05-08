package logger

import (
	"os"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/sirupsen/logrus"
)

// Logger is a wrapper around logrus.Logger
type Logger struct {
	*logrus.Logger
}

// Config holds the configuration for the logger
type Config struct {
	Verbosity string
	Rate      int
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
func New(config Config) *Logger {
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

	return &Logger{
		Logger: logger,
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

	// Create log entry with random fields
	logEntry := l.WithFields(logrus.Fields{
		"service":     service,
		"user_id":     userID,
		"http_method": httpMethod,
		"status_code": statusCode,
		"latency_ms":  latency,
		"ip_address":  ipAddress,
		"timestamp":   time.Now().UnixNano(),
	})

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

// GenerateRandomErrorLog generates a random error log entry
func (l *Logger) GenerateRandomErrorLog() {
	// Generate fake data
	errorMessage := gofakeit.SentenceSimple()
	service := gofakeit.AppName()
	requestID := gofakeit.UUID()
	errorCode := gofakeit.Number(400, 599)
	stackTrace := gofakeit.LoremIpsumSentence(5)

	// Create log entry with random fields
	logEntry := l.WithFields(logrus.Fields{
		"service":     service,
		"request_id":  requestID,
		"error_code":  errorCode,
		"stack_trace": stackTrace,
		"timestamp":   time.Now().UnixNano(),
	})

	logEntry.Error(errorMessage)
}
