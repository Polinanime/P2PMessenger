# P2PMessenger/Dockerfile
FROM golang:1.22.4-alpine

WORKDIR /app

# Copy go mod and sum files
COPY src/go.mod src/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY src/ ./

# Build the application
RUN go build -o messenger

# Command to run the application
CMD ["./messenger"]
