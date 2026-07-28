ARG GO_VERSION=1.24.4
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /app
COPY backend/go.mod backend/go.sum ./backend/
WORKDIR /app/backend
RUN go mod download && go mod verify
WORKDIR /app
COPY . .
RUN ls -la /app
WORKDIR /app/backend

RUN go build -v -o /run-app .

FROM debian:bookworm-slim

# Set working directory to root
WORKDIR /app
COPY --from=builder /run-app /usr/local/bin/

# Copy the app directory
COPY --from=builder /app /app

# Required for secure connections to external services
RUN apt-get update && apt-get install -y ca-certificates

RUN ls -la /app

CMD ["run-app"]
