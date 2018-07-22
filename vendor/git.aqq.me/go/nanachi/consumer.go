package nanachi

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"git.aqq.me/go/retrier"
	"github.com/streadway/amqp"
)

// Consume starts delivering queued messages.
func (c *Consumer) Consume() (<-chan *Delivery, error) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.closed {
		panic("can't consume on closed consumer")
	}

	if c.msgs != nil {
		return c.msgs, nil
	}

	c.msgs = make(chan *Delivery, c.config.PrefetchCount)
	c.stop = make(chan struct{})
	c.canceled = false

	c.run()

	return c.msgs, nil
}

// Cancel method stops deliveries to the consumer channel.
func (c *Consumer) Cancel() {
	c.m.Lock()

	if c.canceled {
		c.m.Unlock()
		return
	}

	close(c.stop)
	c.canceled = true

	c.m.Unlock()

	c.cwg.Wait()

	close(c.msgs)
	c.msgs = nil
}

// Close method performs correct closure of the consumer.
func (c *Consumer) Close() {
	c.Cancel()

	c.m.Lock()

	if c.closed {
		c.m.Unlock()
		return
	}

	if c.channel != nil {
		c.channel.Close()
	}

	c.client.releaseConsumer(c.id)
	c.closed = true

	c.m.Unlock()

	c.wg.Wait()

	close(c.cancels)
	close(c.closes)
}

func (c *Consumer) run() {
	c.wg.Add(1)
	c.cwg.Add(1)

	go func() {
		defer func() {
			c.wg.Done()
			c.cwg.Done()
		}()

		var crName string

		for {
			c.client.retrier.Do(
				func() retrier.Status {
					err := c.consume(crName)

					if err != nil {
						c.notifyError(err)
						return retrier.NeedRetry
					}

					return retrier.Succeed
				},
			)

			select {
			case <-c.stop:
				c.cancel()
				return
			case crName = <-c.cancels:
			case <-c.closes:
			}
		}
	}()
}

func (c *Consumer) consume(crName string) error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.closed {
		panic("can't consume on closed consumer")
	}

	if c.channel == nil {
		err := c.init()

		if err != nil {
			return err
		}
	}

	if !c.source.declared {
		if c.source.Declare != nil {
			err := c.source.Declare(c.channel)

			if err != nil {
				return err
			}
		}

		c.source.declared = true
	}

	err := c.listenMessages(crName)

	if err != nil {
		return err
	}

	return nil
}

func (c *Consumer) init() error {
	ch, err := c.conn.newChannel()

	if err != nil {
		return err
	}

	c.channel = ch
	err = ch.Qos(c.config.PrefetchCount, 0, false)

	if err != nil {
		return err
	}

	c.listenClose()
	c.listenCancel()

	return nil
}

func (c *Consumer) listenClose() {
	closes := c.channel.NotifyClose(make(chan *amqp.Error, 1))

	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		for err := range closes {
			e := errors.New(err.Reason)
			c.notifyError(e)

			c.abort()
		}
	}()
}

func (c *Consumer) listenCancel() {
	cancels := c.channel.NotifyCancel(make(chan string, 1))

	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		for crName := range cancels {
			c.m.Lock()

			if c.closed {
				c.m.Unlock()
				return
			}

			c.source.declared = false
			c.cancels <- crName

			c.m.Unlock()
		}
	}()
}

func (c *Consumer) listenMessages(crName string) error {
	msgsChanList := make([]<-chan amqp.Delivery, 0, 1)

	if c.source.MaxShard > 0 {
		if crName != "" {
			token := strings.Replace(crName, c.config.ConsumerName+".", "", 1)
			shardNum, _ := strconv.Atoi(token)
			queueName := c.shardedQueueName(int32(shardNum))

			msgs, err := c.channel.Consume(
				queueName,
				crName,
				c.config.AutoAck,
				c.config.Exclusive,
				c.config.NoLocal,
				false,
				nil,
			)

			if err != nil {
				return err
			}

			msgsChanList = append(msgsChanList, msgs)
		} else {
			for i := int32(0); i <= c.source.MaxShard; i++ {
				queueName := c.shardedQueueName(i)
				crName := c.shardedConsumerName(i)

				msgs, err := c.channel.Consume(
					queueName,
					crName,
					c.config.AutoAck,
					c.config.Exclusive,
					c.config.NoLocal,
					false,
					nil,
				)

				if err != nil {
					return err
				}

				msgsChanList = append(msgsChanList, msgs)
			}
		}
	} else {
		msgs, err := c.channel.Consume(
			c.source.Queue,
			c.config.ConsumerName,
			c.config.AutoAck,
			c.config.Exclusive,
			c.config.NoLocal,
			false,
			nil,
		)

		if err != nil {
			return err
		}

		msgsChanList = append(msgsChanList, msgs)
	}

	for _, msgs := range msgsChanList {
		c.wg.Add(1)
		c.cwg.Add(1)

		go func(msgs <-chan amqp.Delivery) {
			defer func() {
				c.wg.Done()
				c.cwg.Done()
			}()

			for msg := range msgs {
				var data interface{}
				var err error

				if c.config.ParseBody != nil {
					data, err = c.config.ParseBody(msg.Body)
				}

				c.msgs <- &Delivery{
					Delivery:   msg,
					ParsedBody: data,
					ParseErr:   err,
				}
			}
		}(msgs)
	}

	return nil
}

func (c *Consumer) abort() {
	c.m.Lock()
	defer c.m.Unlock()

	if c.closed {
		return
	}

	c.channel = nil
	c.source.declared = false

	c.closes <- struct{}{}
}

func (c *Consumer) cancel() {
	c.m.Lock()
	defer c.m.Unlock()

	if c.channel != nil {
		if c.source.MaxShard > 0 {
			for i := int32(0); i <= c.source.MaxShard; i++ {
				crName := c.shardedConsumerName(i)
				c.channel.Cancel(crName, false)
			}
		} else {
			c.channel.Cancel(c.config.ConsumerName, false)
		}
	}
}

func (c *Consumer) notifyError(err error) {
	if c.config.ErrorNotifier != nil {
		c.config.ErrorNotifier.Notify(err)
	}
}

func (c *Consumer) shardedQueueName(shardNum int32) string {
	return fmt.Sprintf("%s.%d", c.source.Queue, shardNum)
}

func (c *Consumer) shardedConsumerName(shardNum int32) string {
	return fmt.Sprintf("%s.%d", c.config.ConsumerName, shardNum)
}
