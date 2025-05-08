# Log Genie

A Go application that generates artificial logs for testing purposes of log ingestion systems.

## Features

- Configurable log verbosity (via command line flags and environment variables)
- Configurable log generation rate
- High throughput log generation
- Docker support

## Usage

### Command Line Options

```
Usage of log-genie:
  -rate int
        Number of logs per second (default 10)
  -verbosity string
        Log verbosity level: debug, info, warn, error (default "info")
```

### Environment Variables

- `LOG_GENIE_VERBOSITY`: Log verbosity level (debug, info, warn, error)
- `LOG_GENIE_RATE`: Number of logs per second

### Docker

```bash
docker run -e LOG_GENIE_RATE=100 -e LOG_GENIE_VERBOSITY=debug ghcr.io/rjonczy/log-genie:latest
```

## Building

```bash
# Build locally
go build -o log-genie

# Build Docker image
docker build -t log-genie .
```
