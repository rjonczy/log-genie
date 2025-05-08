package loggenie

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rjonczy/log-genie/pkg/logger"
)

const (
	defaultRate      = 10
	defaultVerbosity = "info"
)

// Main is the entry point for the application
func Main() {
	// Parse command line flags
	rate := flag.Int("rate", defaultRate, "Number of logs per second")
	verbosity := flag.String("verbosity", defaultVerbosity, "Log verbosity level: debug, info, warn, error")
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

	// Create logger
	config := logger.Config{
		Verbosity: *verbosity,
		Rate:      *rate,
	}
	log := logger.New(config)

	// Calculate the interval between log emissions
	interval := time.Second / time.Duration(*rate)

	// Setup ticker for regular logs
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Setup signal catching
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Log startup message
	startupLog := log.WithField("app", "log-genie")
	startupLog.Info(fmt.Sprintf("Starting log generation at %d logs per second with %s verbosity", *rate, *verbosity))

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
