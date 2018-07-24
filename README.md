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
