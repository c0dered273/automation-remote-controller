# syntax=docker/dockerfile:1

FROM golang:1.20-alpine AS builder
WORKDIR /build
COPY . .
RUN go mod download
RUN go build -o user-account-api /build/cmd/user-account-api/main.go

FROM alpine
WORKDIR /
COPY --from=builder /build/user-account-api /user-account-api
EXPOSE 8080
ENTRYPOINT ["/user-account-api"]

