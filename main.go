package main

import (
	"sync"

	"git.aqq.me/go/app/appconf"
	"git.aqq.me/go/app/launcher"
	"github.com/iph0/conf/envconf"
	"github.com/iph0/conf/fileconf"
	"github.com/kak-tus/corrie/reader"
	"github.com/kak-tus/corrie/writer"
	"github.com/kak-tus/healthcheck"
)

var rdr *reader.Reader
var wrt *writer.Writer

func init() {
	fileLdr, err := fileconf.NewLoader("etc")
	if err != nil {
		panic(err)
	}

	envLdr := envconf.NewLoader()

	appconf.RegisterLoader("file", fileLdr)
	appconf.RegisterLoader("env", envLdr)

	appconf.Require("file:corrie.yml")
	appconf.Require("env:^CORRIE_")
}

func main() {
	launcher.Run(func() error {
		healthcheck.Add("/healthcheck", func() (healthcheck.State, string) {
			return healthcheck.StatePassing, "ok"
		})

		rdr = reader.GetReader()

		wrt = writer.GetWriter()
		go wrt.Start()

		healthcheck.Add("/status", status)

		return nil
	})
}

func status() (healthcheck.State, string) {
	var wg sync.WaitGroup
	wg.Add(2)

	var rs bool
	go func() {
		rs = rdr.IsAccessible()
	}()

	var ws bool
	go func() {
		ws = wrt.IsAccessible()
	}()

	wg.Wait()

	if rs && ws {
		return healthcheck.StatePassing, "ok"
	}

	return healthcheck.StateWarning, "nok"
}
