package main

import (
	"log"
	"os"

	"code.cloudfoundry.org/envoy-nginx/app"
	"github.com/pivotal-cf/jhanda"
)

type flags struct {
	Config string `short:"c" default:"C:\\etc\\cf-assets\\envoy_config\\envoy.yaml"`
}

func main() {
	var f flags

	_, err := jhanda.Parse(&f, os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	app.Envoy(f.Config)
}
