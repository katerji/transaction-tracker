# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download || true

# Copy source code
COPY *.go ./
COPY dashboard.html ./

# Build the application with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o transaction-tracker .

# Runtime stage
FROM alpine:latest

# Install SQLite runtime library
RUN apk --no-cache add ca-certificates tzdata sqlite-libs

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/transaction-tracker .

# Create directory for database
RUN mkdir -p /data

# Expose port
EXPOSE 8080

# Run the application
CMD ["./transaction-tracker"]
