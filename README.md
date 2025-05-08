# Log Genie

A Go application that generates artificial logs for testing purposes of log ingestion systems.

## Features

- Configurable log verbosity (via command line flags and environment variables)
- Configurable log generation rate
- High throughput log generation
- Docker support
- OpenTelemetry integration for shipping logs to external collectors

## Usage

### Command Line Options

```
Usage of log-genie:
  -rate int
        Number of logs per second (default 10)
  -verbosity string
        Log verbosity level: debug, info, warn, error (default "info")
  -telemetry
        Enable OpenTelemetry logs export (default false)
  -telemetry-endpoint string
        OpenTelemetry collector endpoint (default "http://localhost:8080/logs")
  -local-logs
        Enable local logs to stdout/stderr even when telemetry is enabled (default true)
```

### Environment Variables

- `LOG_GENIE_VERBOSITY`: Log verbosity level (debug, info, warn, error)
- `LOG_GENIE_RATE`: Number of logs per second
- `LOG_GENIE_TELEMETRY`: Enable OpenTelemetry logs export (true/false or 1/0)
- `LOG_GENIE_TELEMETRY_ENDPOINT`: OpenTelemetry collector endpoint
- `LOG_GENIE_LOCAL_LOGS`: Enable local logs to stdout/stderr even when telemetry is enabled (true/false or 1/0)

### Docker

```bash
# Run with local logging only
docker run ghcr.io/rjonczy/log-genie:latest

# Run with high log rate
docker run -e LOG_GENIE_RATE=100 -e LOG_GENIE_VERBOSITY=debug ghcr.io/rjonczy/log-genie:latest

# Run with OpenTelemetry export
docker run -e LOG_GENIE_TELEMETRY=true -e LOG_GENIE_TELEMETRY_ENDPOINT=collector:4318 ghcr.io/rjonczy/log-genie:latest

# Run with OpenTelemetry export but without local logs
docker run -e LOG_GENIE_TELEMETRY=true -e LOG_GENIE_LOCAL_LOGS=false -e LOG_GENIE_TELEMETRY_ENDPOINT=collector:4318 ghcr.io/rjonczy/log-genie:latest
```

## Building

```bash
# Build locally
go build -o log-genie

# Build Docker image
docker build -t log-genie .
```

## OpenTelemetry Integration

Log Genie can send logs to an OpenTelemetry collector using the OTLP HTTP protocol. This enables integration with various log aggregation systems that support OpenTelemetry.

When telemetry is enabled:
- Logs are sent to the specified OpenTelemetry collector endpoint
- Local logs can be disabled to reduce noise in the console
- A periodic summary is printed showing how many logs have been sent to the collector
- If the collector connection fails, errors are reported in the local logs

To use this feature, you'll need an OpenTelemetry collector running. Example:

```bash
# Run OpenTelemetry Collector using Docker
docker run -p 4318:4318 otel/opentelemetry-collector-contrib:latest
```

Example collector configuration (otel-collector-config.yaml):

```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 1s
    send_batch_size: 1024

exporters:
  logging:
    loglevel: debug
  # Add other exporters like elasticsearch, prometheus, etc.

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging]
```
