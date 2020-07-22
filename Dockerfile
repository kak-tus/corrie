FROM golang:1.10.3-alpine3.8 AS build

WORKDIR /go/src/github.com/kak-tus/corrie

COPY message ./message
COPY reader ./reader
COPY vendor ./vendor
COPY writer ./writer
COPY main.go .

RUN go install

FROM alpine:3.8

COPY --from=build /go/bin/corrie /usr/local/corrie
COPY etc /etc/

RUN adduser -DH user

USER user

ENV \
  CORRIE_RABBITMQ_ADDR= \
  CORRIE_RABBITMQ_VHOST= \
  CORRIE_RABBITMQ_USER= \
  CORRIE_RABBITMQ_PASSWORD= \
  CORRIE_RABBITMQ_MAXRETRY=0 \
  \
  CORRIE_CLICKHOUSE_ADDR= \
  CORRIE_CLICKHOUSE_ALTADDRS= \
  \
  CORRIE_BATCH=1000

CMD ["/usr/local/corrie"]
