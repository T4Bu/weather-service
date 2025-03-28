FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /weather-service

# Use a small alpine image for the final container
FROM alpine:3.19

WORKDIR /

# Copy the binary from builder stage
COPY --from=builder /weather-service /weather-service
COPY --from=builder /app/config.json /config.json

# Expose the application port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/weather-service"] 