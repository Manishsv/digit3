# syntax=docker/dockerfile:1.4

# --- Build stage ---
FROM golang:1.22.3-alpine AS builder
WORKDIR /app
#COPY go.mod go.sum ./
COPY code/services/account/go.mod code/services/account/go.sum ./
RUN go mod download
#COPY . .
COPY code/services/account/ ./
# Build for ARM64 architecture (matches your cluster nodes)
RUN CGO_ENABLED=0 GOOS=linux  go build -o account cmd/server/main.go

# --- Final image ---
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/account ./account
# Copy the realm config file to multiple locations to ensure it's found
#COPY internal/keycloak/realm_config.json ./realm_config.json
COPY code/services/account/internal/keycloak/realm_config.json ./realm_config.json
# Also copy to the internal structure for compatibility
RUN mkdir -p internal/keycloak
#COPY internal/keycloak/realm_config.json ./internal/keycloak/realm_config.json
COPY code/services/account/internal/keycloak/realm_config.json ./internal/keycloak/realm_config.json
#COPY db ./db
#COPY code/services/account/db ./db
EXPOSE 8080
ENTRYPOINT ["./account"] 
