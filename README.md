# Corrie

Reliable (with RabbitMQ) Clickhouse writer.

Notice: if you need performance, it is better idea to use [Ruthie](https://github.com/kak-tus/ruthie) - based on Redis Cluster it has much more better performance, then Corrie, based on RabbitMQ.

## Configuration

### CORRIE_RABBITMQ_ADDR

RabbitMQ address in host:port format

```
CORRIE_RABBITMQ_ADDR=rabbitmq.example.com:5672
```

### CORRIE_RABBITMQ_VHOST

RabbitMQ virtual host to store queue

```
CORRIE_RABBITMQ_VHOST=corrie
```

### CORRIE_RABBITMQ_USER, CORRIE_RABBITMQ_PASSWORD

RabbitMQ user and password

```
CORRIE_RABBITMQ_USER=corrie
CORRIE_RABBITMQ_PASSWORD=somepassword
```

### CORRIE_CLICKHOUSE_ADDR

Primary ClickHouse address in host:port form

```
CORRIE_CLICKHOUSE_ADDR=clickhouse1.example.com:9000
```

### CORRIE_CLICKHOUSE_ALTADDRS

Comma separated list of alternative ClickHouse addresses to loadbalancing. Can be empty

```
CORRIE_CLICKHOUSE_ALTADDRS=clickhouse2.example.com:9000
```

### CORRIE_BATCH

Set batch size of ClickHouse writes.

```
CORRIE_BATCH=10000
```

## Run

```
docker run --rm -it kaktuss/corrie
```

## Write data

To write data use [message](https://godoc.org/github.com/kak-tus/corrie/message) package.

You can write data with nanachi RabbitMQ client (see example) or with any other client.

Pay attention, that Corrie uses sharded queue (with nanachi) hardcoded to use 3 shards. Shards count will be configurable later.
