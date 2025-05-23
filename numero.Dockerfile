FROM golang:1.23-alpine3.21 AS dependencies

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

RUN ./build.sh && go build -o update-kms-keys ./cmd/update-kms-keys

WORKDIR /dist

RUN ls /build

RUN cp /build/main . && cp /build/update-kms-keys .

FROM alpine:3.15 as dist
COPY custom_account_setup_emulator.cdc ./
COPY custom_account_setup_qa.cdc ./
COPY custom_account_setup_staging.cdc ./
COPY custom_account_setup_production.cdc ./
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /dist/main /
COPY --from=builder /dist/update-kms-keys /

CMD FLOW_WALLET_PORT=$PORT FLOW_WALLET_DATABASE_DSN=$DATABASE_URL FLOW_WALLET_IDEMPOTENCY_MIDDLEWARE_REDIS_URL=$REDIS_URL /main