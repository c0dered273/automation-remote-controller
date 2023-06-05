# syntax=docker/dockerfile:1

FROM golang:1.20-bullseye AS builder
WORKDIR /build
RUN apt update && apt install -y --no-install-recommends libpcap-dev

COPY . .
RUN go mod download
RUN go build -o remote-control-client /build/cmd/remote-control-client/main.go

FROM alpine
WORKDIR /
COPY --from=builder /build/remote-control-client /remote-control-client
ENTRYPOINT ["/remote-control-client"]

