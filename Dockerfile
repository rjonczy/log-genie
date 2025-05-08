# Build stage
FROM golang:1.21-alpine AS build

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

# Run the application
ENTRYPOINT ["/log-genie"]
