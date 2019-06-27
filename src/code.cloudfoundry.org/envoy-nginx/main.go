package main

import (
	"log"
	"os"

	"code.cloudfoundry.org/envoy-nginx/app"
	"github.com/pivotal-cf/jhanda"
)

type flags struct {
	Config        string `short:"c" default:"C:\\etc\\cf-assets\\envoy_config\\envoy.yaml"`
	SdsCreds      string `short:"k" default:"C:\\etc\\cf-assets\\envoy_config\\sds-server-cert-and-keys.yaml"`
	SdsValidation string `short:"v" default:"C:\\etc\\cf-assets\\envoy_config\\sds-server-validation-context.yml"`
}

func main() {
	var f flags

	_, err := jhanda.Parse(&f, os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	logger := app.NewLogger(os.Stdout)
	application := app.NewApp(logger, f.Config)

	err = application.Load(f.SdsCreds, f.SdsValidation)
	if err != nil {
		log.Fatal(err)
	}
}
