package reader

import (
	"git.aqq.me/go/nanachi"
	"go.uber.org/zap"
)

// Reader hold object
type Reader struct {
	logger   *zap.SugaredLogger
	config   readerConfig
	nanachi  *nanachi.Client
	consumer *nanachi.Consumer
	C        <-chan *nanachi.Delivery
}

type readerConfig struct {
	Rabbit rabbitConfig
	Batch  int
}

type rabbitConfig struct {
	URI         string
	Queue       string
	QueueFailed string
	MaxShard    int
}
