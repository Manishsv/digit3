# -------------------------
# Build stage
# -------------------------
    FROM golang:1.24-alpine AS build

    # Accept arguments
    ARG WORK_DIR=.
    ARG MAIN_PACKAGE=./cmd/server
    
    # Set binary name to always be "app"
    ENV BINARY_NAME=app
    WORKDIR /app
    
    # Copy go.mod and go.sum first for caching
    COPY ${WORK_DIR}/go.mod ${WORK_DIR}/go.sum ./
    RUN go mod download
    
    # Copy source code
    COPY ${WORK_DIR} .
    
    # Build the binary
    RUN CGO_ENABLED=0 GOOS=linux go build -o ${BINARY_NAME} ${MAIN_PACKAGE}
    
    # -------------------------
    # Runtime stage
    # -------------------------
    FROM alpine:3.19
    
    WORKDIR /opt/digit
    
    # Use same binary name "app"
    ENV BINARY_NAME=app
    
    # Copy binary only
    COPY --from=build /app/${BINARY_NAME} .
    
    # Copy migrations
    COPY db/migrations .db/migrations
    
    # Set permissions
    RUN chmod +x /opt/digit/${BINARY_NAME}
    
    EXPOSE 8080
    ENTRYPOINT ["./app"]