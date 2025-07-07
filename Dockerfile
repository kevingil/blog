ARG GO_VERSION=1.24.4
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN ls -la /app
WORKDIR /app

RUN go build -v -o /run-app ./backend

FROM debian:bookworm-slim

# Set working directory to root
WORKDIR /app
COPY --from=builder /run-app /usr/local/bin/

# Copy the app directory
COPY --from=builder /app /app
RUN apt-get update && apt-get install -y ca-certificates

RUN ls -la /app

CMD ["run-app"]
