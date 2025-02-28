# Use Go 1.23.0 as the builder image
FROM golang:1.23.0 AS builder

# Set working directory in the container
WORKDIR /app

# Copy go.work file (to ensure the workspace is configured correctly)
COPY go.work* ./

# Copy the go.mod and go.sum for all services
COPY ./api-gateway/go.mod ./api-gateway/
COPY ./api-gateway/go.sum* ./api-gateway/
COPY ./user-service/go.mod ./user-service/
COPY ./user-service/go.sum* ./user-service/
COPY ./messaging/go.mod ./messaging/
COPY ./messaging/go.sum* ./messaging/
# Download dependencies for all modules
RUN go work sync

# Copy the source code for each service
COPY ./user-service/ ./user-service/
COPY ./api-gateway/ ./api-gateway/
COPY ./messaging/ ./messaging/
# Build the application (user-service in this case)
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux go build -o userServiceApp ./user-service/cmd/app && chmod +x userServiceApp

# Final stage: Build a smaller image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates
RUN apk update && apk add --no-cache curl
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/userServiceApp ./

# Run the application
CMD ["./userServiceApp"]
