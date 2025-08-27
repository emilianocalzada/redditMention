# Stage 1: Build
FROM golang:1.25 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o myapp main.go

# Stage 2: Runtime
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/myapp /app/myapp
VOLUME ["/app/pb_data/"]
EXPOSE 8080
ENTRYPOINT ["/app/myapp", "serve", "--http=0.0.0.0:8080"]