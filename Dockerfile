# Use multi-stage build to keep the final image small
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install tzdata
RUN apk add --no-cache tzdata
# **Important:** Copy go.mod and go.sum *first*
COPY go.mod go.sum ./
# Download dependencies *before* copying the rest of the source code.
# This allows Docker to cache this layer if only the source code changes.
RUN go mod download

# Copy the entire project source code
COPY . .

# Generate Swagger documentation
RUN swag init

# Build the application.  Disable cgo for smaller, simpler images, if it's not needed.
RUN CGO_ENABLED=0 go build -o main .

# Create a minimal final image
FROM alpine:3.18

# Copy everything from the build stage to ensure migrations and other necessary files are included
COPY --from=builder /app /app
COPY --from=builder /app/main /main

# Expose the port your application listens on
EXPOSE 8080

# Set the entrypoint for the container
CMD ["/main"]
