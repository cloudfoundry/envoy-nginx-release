package main

import (
	"fmt"
	"log"
	"os"

	"code.cloudfoundry.org/envoy-nginx/app"
	flags "github.com/jessevdk/go-flags"
)

type options struct {
	Config        string `short:"c"         default:"C:\\etc\\cf-assets\\envoy_config\\envoy.yaml"`
	SdsCreds      string `long:"creds"      default:"C:\\etc\\cf-assets\\envoy_config\\sds-server-cert-and-keys.yaml"`
	SdsValidation string `long:"validation" default:"C:\\etc\\cf-assets\\envoy_config\\sds-server-validation-context.yml"`
}

func main() {
	var o options

	p := flags.NewParser(&o, flags.IgnoreUnknown)
	_, err := p.ParseArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("Flags parse: %s", err)
	}

	logger := app.NewLogger(os.Stdout)
	application := app.NewApp(logger, o.Config)

	err = application.Load(o.SdsCreds, o.SdsValidation)
	if err != nil {
		log.Fatalf("Application load: %s", err)
	}
}
