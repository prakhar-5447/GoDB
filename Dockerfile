# Stage 1: Build the binary using a Golang base image.
FROM golang:1.24-alpine AS builder

# Install build tools including gcc
RUN apk update && apk add --no-cache build-base git

WORKDIR /app

# Copy go.mod and go.sum first to leverage caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Enable CGO and build the binary
ENV CGO_ENABLED=1
RUN go build -o godb-server ./cmd/server/main.go

# Stage 2: Create a minimal runtime image.
FROM alpine:latest

# Install ca-certificates (if needed)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Create a directory for database data
RUN mkdir -p /root/data

# Copy the built binary from the builder stage.
COPY --from=builder /app/godb-server .

# Expose the gRPC port.
EXPOSE 50051

# Set environment variable for DB data directory.
ENV DB_DATA_DIR=/root/data

# Run the binary.
CMD ["./godb-server"]
