# Build stage
FROM golang:1.23-alpine AS builder

# Install dependencies (including SQLite development files)
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -o gds ./cmd/main.go

# Final stage using alpine (since we need SQLite libraries)
FROM alpine:latest

# Install SQLite runtime
RUN apk add --no-cache sqlite

# Copy the binary from the builder stage
COPY --from=builder /app/gds /app/gds

# Copy database migrations
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/static /app/static
COPY --from=builder /app/github_data.db /app/github_data.db

# Copy any required files (like .env if needed)
# COPY --from=builder /app/.env /app/.env

WORKDIR /app

EXPOSE 8080

# Command to run the application
CMD ["/app/gds"]
