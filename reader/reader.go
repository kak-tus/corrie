package reader

import (
	"fmt"
	"time"

	"git.aqq.me/go/app/appconf"
	"git.aqq.me/go/app/applog"
	"git.aqq.me/go/app/event"
	"git.aqq.me/go/nanachi"
	"git.aqq.me/go/retrier"
	"github.com/mitchellh/mapstructure"
	"github.com/streadway/amqp"
)

var rdr *Reader

func init() {
	event.Init.AddHandler(
		func() error {
			cnfMap := appconf.GetConfig()["reader"]

			var cnf readerConfig
			err := mapstructure.Decode(cnfMap, &cnf)
			if err != nil {
				return err
			}

			logger := applog.GetLogger()

			nn, err := nanachi.NewClient(
				nanachi.ClientConfig{
					URI:           cnf.Rabbit.URI,
					Heartbeat:     time.Second * 15,
					ErrorNotifier: new(Reader),

					RetrierConfig: &retrier.Config{
						RetryPolicy: []time.Duration{time.Second},
					},
				},
			)
			if err != nil {
				return err
			}

			rdr = &Reader{
				logger:  logger,
				config:  cnf,
				nanachi: nn,
			}

			return nil
		},
	)

	event.Stop.AddHandler(
		func() error {
			rdr.consumer.Close()
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
	src := &nanachi.Source{
		Queue:    r.config.Rabbit.Queue,
		MaxShard: int32(r.config.Rabbit.MaxShard),
		Declare:  rdr.declare,
	}

	cns := r.nanachi.NewConsumer(
		nanachi.ConsumerConfig{
			Source:        src,
			PrefetchCount: r.config.Batch,
		},
	)

	r.consumer = cns

	msgs, err := r.consumer.Consume()

	if err != nil {
		r.logger.Error(err)
		return
	}

	r.C = msgs

	// for msg := range msgs {
	// 	fmt.Println("Consumed message:", string(msg.Body))
	// }

	return
}

// Notify nanachi method
func (r Reader) Notify(err error) {
	// r.logger.Error(err)
	println(err.Error())
}

func (r Reader) declare(ch *amqp.Channel) error {
	for i := 0; i <= r.config.Rabbit.MaxShard; i++ {
		shardName := fmt.Sprintf("%s.%d", r.config.Rabbit.Queue, i)

		_, err := ch.QueueDeclare(shardName, true, false, false, false, nil)

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
