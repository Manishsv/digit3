# syntax=docker/dockerfile:1.4

# --- Build stage ---
    FROM golang:1.23-alpine AS builder
    WORKDIR /app
    COPY go.mod go.sum ./
    RUN go mod download
    COPY . .
    RUN CGO_ENABLED=0 GOOS=linux go build -o boundary ./cmd/server/main.go
    
    # --- Final image ---
    FROM alpine:3.19
    WORKDIR /app
    COPY --from=builder /app/boundary ./boundary
    COPY db ./db
    EXPOSE 8080
    ENTRYPOINT ["./boundary"]