ARG GO_VERSION=1.23.0
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN ls -la /app
WORKDIR /app
RUN go build -v -o /run-app ./backend

FROM debian:bookworm

# Set working directory to root where env file should be
WORKDIR /app
COPY --from=builder /run-app /usr/local/bin/
# Copy the entire app directory to preserve env files
COPY --from=builder /app /app
RUN apt-get update && apt-get install -y ca-certificates
# List files to verify env file is present
RUN ls -la /app
CMD ["run-app"]
