FROM golang:1.22-alpine

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire project
COPY . .

# Build the application
RUN go build -o p2pmessenger ./cmd/p2pmessenger

ENTRYPOINT ["/app/scripts/entrypoint.sh"]
