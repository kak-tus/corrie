package writer

import (
	"database/sql"

	"go.uber.org/zap"
)

// Writer hold object
type Writer struct {
	logger *zap.SugaredLogger
	config writerConfig
	db     *sql.DB
}

type writerConfig struct {
	ClickhouseURI string
}
