FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o weather-service ./cmd/weather-service

# Use a minimal alpine image for the final container
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/weather-service .
COPY --from=builder /app/config.json .

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./weather-service"] 