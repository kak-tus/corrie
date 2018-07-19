package main

import (
	"git.aqq.me/go/app/appconf"
	"git.aqq.me/go/app/launcher"
	"github.com/iph0/conf/envconf"
	"github.com/iph0/conf/fileconf"
	"github.com/kak-tus/corrie/reader"
	"github.com/kak-tus/corrie/writer"
	"github.com/kak-tus/healthcheck"
)

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
	var rdr *reader.Reader
	var wrt *writer.Writer

	launcher.Run(func() error {
		healthcheck.Add("/healthcheck", func() (healthcheck.State, string) {
			return healthcheck.StatePassing, "ok"
		})

		rdr = reader.GetReader()
		go rdr.Start()

		wrt = writer.GetWriter()
		go wrt.Start()

		return nil
	})
}
