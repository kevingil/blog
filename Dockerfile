# Go builder
FROM golang:1.22.5 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o app


# Final package
FROM alpine:3.18.4
WORKDIR /app
COPY --from=builder /app .

RUN apk update && apk add ca-certificates

# Run app
EXPOSE 8080
CMD ["./app"]
