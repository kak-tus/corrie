package writer

import (
	"database/sql"
	"sync"

	"git.aqq.me/go/nanachi"
	"git.aqq.me/go/retrier"
	jsoniter "github.com/json-iterator/go"
	"github.com/kak-tus/corrie/message"
	"github.com/kak-tus/corrie/reader"
	"go.uber.org/zap"
)

// Writer hold object
type Writer struct {
	logger     *zap.SugaredLogger
	config     writerConfig
	db         *sql.DB
	c          <-chan *nanachi.Delivery
	decoder    jsoniter.API
	m          *sync.Mutex
	reader     *reader.Reader
	toSendVals map[string][]toSend
	toSendCnts map[string]int
	retrier    *retrier.Retrier
}

type writerConfig struct {
	ClickhouseURI string
	Batch         int
	Period        int
}

type toSend struct {
	parsed  message.Message
	nanachi *nanachi.Delivery
}
