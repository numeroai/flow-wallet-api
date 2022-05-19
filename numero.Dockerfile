FROM golang:1.17.6-alpine3.15 AS dependencies

RUN apk update && apk add --no-cache \
  ca-certificates \
  musl-dev \
  gcc \
  build-base \
  git

ENV GO111MODULE=on \
  CGO_ENABLED=1 \
  GOOS=linux \
  GOARCH=amd64

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

FROM dependencies AS builder

COPY . .

RUN ./build.sh

WORKDIR /dist

RUN cp /build/main .

FROM alpine:3.15 as dist
COPY custom_account_setup.cdc ./
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /dist/main /

CMD FLOW_WALLET_PORT=$PORT FLOW_WALLET_DATABASE_DSN=$DATABASE_URL /main