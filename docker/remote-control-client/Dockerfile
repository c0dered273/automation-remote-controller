# syntax=docker/dockerfile:1

FROM golang:1.20-alpine AS builder
WORKDIR /build
COPY . .
RUN go mod download
RUN go build -o remote-control-client /build/cmd/remote-control-client/main.go

FROM alpine
WORKDIR /
COPY --from=builder /build/remote-control-client /remote-control-client
EXPOSE 8080
ENTRYPOINT ["/remote-control-client"]

