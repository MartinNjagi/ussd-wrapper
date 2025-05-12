# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies including swag
RUN apk add --no-cache tzdata git

# Install swag CLI tool
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.4

# Copy go.mod and go.sum separately to leverage caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and generate Swagger docs
COPY . .

# Run swag init only if needed
RUN /go/bin/swag init

# Build the Go binary
RUN CGO_ENABLED=0 go build -o main .

# Stage 2: Final image
FROM alpine:3.18

WORKDIR /app

# Install tzdata and set timezone to Africa/Nairobi
RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Africa/Nairobi /etc/localtime && \
    echo "Africa/Nairobi" > /etc/timezone
ENV TZ=Africa/Nairobi

# Copy the built binary and necessary files
COPY --from=builder /app /app
COPY --from=builder /app/main /main

EXPOSE 8080
CMD ["/main"]
