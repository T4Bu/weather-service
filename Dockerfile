FROM golang:1.19-alpine AS builder

# Set the working directory
WORKDIR /app

# Install dependencies required for building
RUN apk add --no-cache git

# Copy go.mod and go.sum to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o weather-service .

# Create a minimal runtime image
FROM alpine:latest

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/weather-service .

# Copy configuration files
COPY config.json .
COPY test_config.json .

# Set ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose the API port
EXPOSE 8080

# Define environment variables with default values
ENV CONFIG_FILE="config.json"
ENV PORT=8080
ENV UPDATE_INTERVAL="5m"
ENV ENABLE_RATE_LIMIT="true"

# Command to run the application
ENTRYPOINT ["sh", "-c", "./weather-service -config=${CONFIG_FILE} -port=${PORT} -update=${UPDATE_INTERVAL} -rate-limit=${ENABLE_RATE_LIMIT}"] 