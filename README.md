# Log-Genie

A tool for generating logs and testing OpenTelemetry logging pipelines.

## Features

- Generate random logs at configurable rates
- Support for local logging and OpenTelemetry export
- Support for resource attributes including `application_id`
- Ability to view responses from the OTEL collector

## Usage

```bash
# Run with local logging only
./log-genie

# Enable OpenTelemetry export
./log-genie --telemetry --telemetry-endpoint=localhost:4318

# Specify application_id
./log-genie --telemetry --telemetry-endpoint=localhost:4318 --application-id=my-app

# Show responses from OTEL collector
./log-genie --telemetry --telemetry-endpoint=localhost:4318 --show-responses
```

## Command Line Flags

| Flag                | Environment Variable         | Default         | Description                                  |
|---------------------|------------------------------|-----------------|----------------------------------------------|
| `--rate`            | `LOG_GENIE_RATE`             | 10              | Number of logs per second                    |
| `--verbosity`       | `LOG_GENIE_VERBOSITY`        | info            | Log level: debug, info, warn, error          |
| `--telemetry`       | `LOG_GENIE_TELEMETRY`        | false           | Enable OpenTelemetry logs export             |
| `--telemetry-endpoint` | `LOG_GENIE_TELEMETRY_ENDPOINT` | collector:4318 | OpenTelemetry collector endpoint            |
| `--local-logs`      | `LOG_GENIE_LOCAL_LOGS`       | false           | Enable local logs when telemetry is enabled  |
| `--show-responses`  | `LOG_GENIE_SHOW_RESPONSES`   | false           | Show responses from the OTEL collector       |
| `--application-id`  | `LOG_GENIE_APPLICATION_ID`   | log-genie       | Application ID for OTEL resource attributes  |

## Testing with Local OTEL Collector

1. Start the local OTEL collector using the provided config:

```bash
# Install the OpenTelemetry Collector if needed
# For example, using Docker:
docker run -p 4317:4317 -p 4318:4318 -v $(pwd)/collector-config.yaml:/etc/otelcol/config.yaml otel/opentelemetry-collector-contrib

# Or using the standalone collector:
otelcol-contrib --config collector-config.yaml
```

2. Run log-genie with and without an application_id:

```bash
# Test without application_id (logs will show in both pipelines)
./log-genie --telemetry --telemetry-endpoint=localhost:4318 --show-responses

# Test with application_id (logs will show in both pipelines)
./log-genie --telemetry --telemetry-endpoint=localhost:4318 --application-id=my-app --show-responses
```

## Local OTEL Config Explanation

The provided `collector-config.yaml` sets up:

1. Standard OTLP receivers on default ports
2. Two log pipelines:
   - A standard pipeline that accepts all logs
   - A pipeline with a filter processor that only accepts logs with an `application_id` attribute

This configuration allows testing and validating that the `application_id` attribute is properly set in the logs.
