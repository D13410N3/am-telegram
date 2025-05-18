# syntax=docker/dockerfile:1.4

FROM golang:1.23.2-alpine AS builder

WORKDIR /app
COPY . .

RUN go build -ldflags="-s -w" -o am-telegram app.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /root/
COPY --from=builder /app/am-telegram .

EXPOSE 8080
ENTRYPOINT ["./am-telegram"]
