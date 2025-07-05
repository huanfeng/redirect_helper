# Build stage
FROM golang:1.22.2-alpine AS builder

WORKDIR /app

# Copy go mod file
COPY go.mod ./

# Download dependencies (if any)
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o redirect_helper ./cmd/redirect_helper

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/redirect_helper .

# Expose port 8001 (default port)
EXPOSE 8001

# Run the binary
CMD ["./redirect_helper", "-server"]