# Build stage
FROM golang:1.23.5-alpine AS builder

# Install git and ca-certificates (needed for go mod download with private repos)
RUN apk add --no-cache git ca-certificates tzdata

# Create appuser for security
RUN adduser -D -g '' appuser

WORKDIR /build

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build the statically linked executable with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o app main.go

# Final stage - using distroless for security and minimal size
FROM gcr.io/distroless/static-debian12:nonroot

# Copy timezone data and certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from builder stage
COPY --from=builder /build/app /app



# Use non-root user (distroless nonroot user has UID 65532)
USER 65532:65532

# Expose port
EXPOSE 8080

# Health check using the binary itself (no curl needed)
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app", "--help"] || exit 1

# Run the binary
ENTRYPOINT ["/app", "serve", "--http=0.0.0.0:8080"]