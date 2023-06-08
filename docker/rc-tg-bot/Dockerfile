# syntax=docker/dockerfile:1

FROM golang:1.20-alpine AS builder
WORKDIR /build
COPY . .
RUN go mod download
RUN go build -o rc-tg-bot /build/cmd/rc-tg-bot/main.go

FROM alpine
WORKDIR /
COPY --from=builder /build/rc-tg-bot /rc-tg-bot
EXPOSE 8081
ENTRYPOINT ["/rc-tg-bot"]

