package reader

import (
	"fmt"
	"time"

	"git.aqq.me/go/app/appconf"
	"git.aqq.me/go/app/applog"
	"git.aqq.me/go/app/event"
	"git.aqq.me/go/nanachi"
	"git.aqq.me/go/retrier"
	"github.com/iph0/conf"
	"github.com/streadway/amqp"
)

var rdr *Reader

func init() {
	event.Init.AddHandler(
		func() error {
			cnfMap := appconf.GetConfig()["reader"]

			var cnf readerConfig
			err := conf.Decode(cnfMap, &cnf)
			if err != nil {
				return err
			}

			rdr = &Reader{
				logger: applog.GetLogger().Sugar(),
				config: cnf,
			}

			rdr.logger.Info("Started reader")

			return nil
		},
	)

	event.Stop.AddHandler(
		func() error {
			rdr.logger.Info("Stop reader")

			rdr.nanachi.Close()

			return nil
		},
	)
}

// GetReader return instance
func GetReader() *Reader {
	return rdr
}

// Start reader
func (r *Reader) Start() {
	client, err := nanachi.NewClient(
		nanachi.ClientConfig{
			URI:           r.config.Rabbit.URI,
			Heartbeat:     time.Second * 15,
			ErrorNotifier: r,
			RetrierConfig: &retrier.Config{
				RetryPolicy: []time.Duration{time.Second},
			},
		},
	)
	if err != nil {
		r.logger.Panic(err)
	}

	r.nanachi = client

	src := &nanachi.Source{
		Queue:    r.config.Rabbit.Queue,
		MaxShard: int32(r.config.Rabbit.MaxShard),
		Declare:  rdr.declare,
	}

	// Prefetch count must be greater then batch, to prevent temporary blocking
	cons := r.nanachi.NewConsumer(
		nanachi.ConsumerConfig{
			Source:        src,
			PrefetchCount: r.config.Batch * 10,
		},
	)

	r.consumer = cons

	msgs, err := r.consumer.Consume()

	if err != nil {
		r.logger.Panic(err)
	}

	r.C = msgs

	dst := &nanachi.Destination{
		RoutingKey: r.config.Rabbit.QueueFailed,
		Declare:    rdr.declare,
	}

	producer := client.NewSmartProducer(
		nanachi.SmartProducerConfig{
			Destinations:      []*nanachi.Destination{dst},
			Mandatory:         true,
			PendingBufferSize: 1000000,
		},
	)

	r.producer = producer
}

// Notify nanachi method
func (r Reader) Notify(err error) {
	r.logger.Error(err)
}

func (r Reader) declare(ch *amqp.Channel) error {
	for i := 0; i <= r.config.Rabbit.MaxShard; i++ {
		shardName := fmt.Sprintf("%s.%d", r.config.Rabbit.Queue, i)

		_, err := ch.QueueDeclare(shardName, true, false, false, false, nil)
		if err != nil {
			return err
		}

		_, err = ch.QueueDeclare(r.config.Rabbit.QueueFailed, true, false, false, false, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// IsAccessible checks RabbitMQ status
func (r Reader) IsAccessible() bool {
	// TODO ping
	return true
}

// Stop reader
func (r Reader) Stop() {
	r.consumer.Cancel()
}

// ToFailedQueue move message to failed queue
func (r Reader) ToFailedQueue(m *nanachi.Delivery) {
	r.producer.Send(
		nanachi.Publishing{
			RoutingKey: r.config.Rabbit.QueueFailed,
			Publishing: amqp.Publishing{
				ContentType:  "text/plain",
				Body:         m.Body,
				DeliveryMode: amqp.Persistent,
			},
		},
	)
}
