package writer

import (
	"database/sql"

	"git.aqq.me/go/nanachi"
	"go.uber.org/zap"
)

// Writer hold object
type Writer struct {
	logger *zap.SugaredLogger
	config writerConfig
	db     *sql.DB
	c      <-chan *nanachi.Delivery
}

type writerConfig struct {
	ClickhouseURI string
}
