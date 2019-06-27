package main

import (
	"log"
	"os"

	"github.com/pivotal-cf/jhanda"
)

type flags struct {
	Config string `short:"c" env:"ENVOY_FILE" default:"C:\\etc\\cf-assets\\envoy_config\\envoy.yaml"`
}

func main() {
	var f flags

	_, err := jhanda.Parse(&f, os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	envoy(f.Config)
}
