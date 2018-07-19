package writer

import (
	"database/sql"
	"fmt"

	"git.aqq.me/go/app/appconf"
	"git.aqq.me/go/app/applog"
	"git.aqq.me/go/app/event"
	"github.com/kshvakov/clickhouse"
	"github.com/mitchellh/mapstructure"
)

var wrt *Writer

func init() {
	event.Init.AddHandler(
		func() error {
			cnfMap := appconf.GetConfig()["writer"]

			var cnf writerConfig
			err := mapstructure.Decode(cnfMap, &cnf)
			if err != nil {
				return err
			}

			logger := applog.GetLogger()

			db, err := sql.Open("clickhouse", cnf.ClickhouseURI)
			if err != nil {
				return err
			}

			err = db.Ping()
			if err != nil {
				exception, ok := err.(*clickhouse.Exception)
				if ok {
					return fmt.Errorf("[%d] %s \n%s", exception.Code, exception.Message, exception.StackTrace)
				}

				return err
			}

			wrt = &Writer{
				logger: logger,
				config: cnf,
				db:     db,
			}

			return nil
		},
	)

	event.Stop.AddHandler(
		func() error {
			wrt.db.Close()
			return nil
		},
	)
}

// GetWriter return instance
func GetWriter() *Writer {
	return wrt
}

// Start writer
func (w Writer) Start() {
	tx, err := w.db.Begin()
	if err != nil {
		w.logger.Error(err)
		return
	}

	stmt, err := tx.Prepare("INSERT INTO bulk.box (dt_part,dt,partner_id) VALUES (?,?,?);")
	if err != nil {
		tx.Rollback()
		w.logger.Error(err)
		return
	}

	_, err = stmt.Exec("2018-07-19", "2018-07-19 21:00:00", 1)
	if err != nil {
		tx.Rollback()
		w.logger.Error(err)
		return
	}
	_, err = stmt.Exec("2018-07-19", "2018-07-19 21:00:00", "z")
	if err != nil {
		tx.Rollback()
		w.logger.Error(err)
		return
	}

	err = tx.Commit()
	if err != nil {
		w.logger.Error(err)
		return
	}

	return
}

func (w Writer) toFailedPool() {
}
