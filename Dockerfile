# Build stage
FROM golang:1.24-alpine AS build

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /log-genie

# Runtime stage
FROM alpine:latest

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /

# Copy binary from build stage
COPY --from=build /log-genie /log-genie

# Default to no telemetry, but allow it to be enabled via env vars
ENV LOG_GENIE_TELEMETRY=false
ENV LOG_GENIE_TELEMETRY_ENDPOINT=collector:4318
ENV LOG_GENIE_LOCAL_LOGS=true

# Application listens on no port, but we document that logs go to stdout/stderr
EXPOSE 80

# Run the application
ENTRYPOINT ["/log-genie"]
