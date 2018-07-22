package launcher

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"git.aqq.me/go/app"
	"git.aqq.me/go/app/applog"
	"git.aqq.me/go/app/event"
	"go.uber.org/zap"
)

var lchr *launcher

type launcher struct {
	logger *zap.SugaredLogger
	reload chan struct{}
	stop   chan struct{}
}

func init() {
	lchr = &launcher{
		reload: make(chan struct{}, 1),
		stop:   make(chan struct{}),
	}

	event.Init.AddHandler(
		func() error {
			lchr.logger = applog.GetLogger()
			return nil
		},
	)
}

// Run method launches an application
func Run(appStart func() error) {
	lchr.run(appStart)
}

func (l *launcher) run(appStart func() error) {
	err := app.Init()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Start failed:", err)
		return
	}

	l.listenSignals()
	err = appStart()

	if err != nil {
		l.logger.Error("Start failed: ", err)
		return
	}

	l.logger.Info("Started")

LOOP:
	for {
		select {
		case <-l.stop:
			break LOOP
		case <-l.reload:
			err := app.Reload()

			if err != nil {
				l.logger.Error("Reload failed: ", err)
				continue LOOP
			}

			l.logger.Info("Reloaded")
		}
	}

	err = app.Stop()

	if err != nil {
		l.logger.Error("Stopped with error: ", err)
		return
	}
}

func (l *launcher) listenSignals() {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	var stopped bool

	go func() {
		for sig := range signals {
			l.logger.Info("Got signal: ", sig)

			if sig == syscall.SIGHUP {
				l.reload <- struct{}{}
			} else {
				if !stopped {
					close(l.stop)
					stopped = true
				}
			}
		}
	}()
}
