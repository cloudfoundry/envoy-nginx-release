package main

import (
	"log"
	"os"

	"code.cloudfoundry.org/envoy-nginx/app"
)

func main() {
	flags := app.NewFlags()
	opts := flags.Parse(os.Args[1:])

	logger := app.NewLogger(os.Stdout)
	application := app.NewApp(logger, opts.EnvoyConfig)

	err := application.Load(opts.SdsCreds, opts.SdsValidation)
	if err != nil {
		log.Fatalf("Application load: %s", err)
	}
}
