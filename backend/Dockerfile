# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies and CA certificates
RUN apk add --no-cache git ca-certificates

# Disable SSL verification for corporate proxy environments
RUN git config --global http.sslVerify false
ENV GOINSECURE="*"
ENV GOPRIVATE="*"
ENV GOPROXY="https://proxy.golang.org,direct"

# Copy source code and go.mod
COPY . .

# Generate go.sum and download dependencies
RUN go mod tidy && go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Final stage
FROM alpine:3.19

WORKDIR /app

# Copy ca-certificates from builder stage (avoids SSL issues)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /app/server .

# Create uploads directory
RUN mkdir -p /app/uploads

# Expose port
EXPOSE 8080

# Run the application
CMD ["./server"]
