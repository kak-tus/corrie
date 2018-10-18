package main

import (
	"fmt"
	"time"

	"git.aqq.me/go/nanachi"
	"git.aqq.me/go/retrier"
	"github.com/kak-tus/corrie/message"
	"github.com/streadway/amqp"
)

func main() {
	client, err := nanachi.NewClient(
		nanachi.ClientConfig{
			URI:           "amqp://example:example@example.com:5672/example",
			Heartbeat:     time.Second * 15,
			ErrorNotifier: new(errorStdoutNotifier),
			RetrierConfig: &retrier.Config{
				RetryPolicy: []time.Duration{time.Second},
			},
		},
	)

	if err != nil {
		panic(err)
	}

	queueName := "messages"
	maxShard := 2

	dst := &nanachi.Destination{
		RoutingKey: queueName,
		MaxShard:   int32(maxShard),
		Declare: func(ch *amqp.Channel) error {
			for i := 0; i <= maxShard; i++ {
				shardName := fmt.Sprintf("%s.%d", queueName, i)

				_, err := ch.QueueDeclare(shardName, true, false, false, false, nil)
				if err != nil {
					panic(err)
				}
			}

			return nil
		},
	}

	producer := client.NewSmartProducer(
		nanachi.SmartProducerConfig{
			Destinations:      []*nanachi.Destination{dst},
			Mandatory:         true,
			PendingBufferSize: 1000000,
			Confirm:           true,
		},
	)

	body, err := message.Message{
		Query: "INSERT INTO default.test (some_field) VALUES (?);",
		Data:  []interface{}{1},
	}.Encode()

	if err != nil {
		panic(err)
	}

	producer.Send(
		nanachi.Publishing{
			RoutingKey: queueName,
			Publishing: amqp.Publishing{
				ContentType:   "text/plain",
				CorrelationId: "1",
				Body:          body,
				DeliveryMode:  amqp.Persistent,
			},
		},
	)

	client.Close()
}
