# Corrie

Reliable (with RabbitMQ) Clickhouse writer.

## Configuration

```
CORRIE_RABBITMQ_ADDR=rabbitmq.example.com:5672
CORRIE_RABBITMQ_VHOST=corrie
CORRIE_RABBITMQ_USER=corrie
CORRIE_RABBITMQ_PASSWORD=somepassword
CORRIE_CLICKHOUSE_ADDR=clickhouse1.example.com:9000
CORRIE_CLICKHOUSE_ALTADDRS=clickhouse2.example.com:9000
```

## Run

```
docker run --rm -it kaktuss/corrie
```

## Write data

To write data use [message](https://godoc.org/github.com/kak-tus/corrie/message) package.

You can write data with nanachi RabbitMQ client (see example) or with any other client.

Pay attention, that Corrie uses sharded queue (with nanachi) hardcoded to use 3 shards. Shards count will be configurable later.
