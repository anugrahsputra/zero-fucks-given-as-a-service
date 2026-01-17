# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies if needed (none strictly required for this simple app, but git is common)
# RUN apk add --no-cache git

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

FROM alpine:3.20

WORKDIR /app

RUN adduser -D appuser

COPY --from=builder /app/server .
COPY zero-fucks.json .

USER appuser

EXPOSE 8080
CMD ["./server"]
