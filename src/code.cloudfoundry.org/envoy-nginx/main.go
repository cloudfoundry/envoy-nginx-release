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
	cmd := app.NewCmd(os.Stdout, os.Stderr)
	application := app.NewApp(logger, cmd, opts.EnvoyConfig)

	nginxPath, err := application.GetNginxPath()
	if err != nil {
		log.Fatalf("envoy-nginx application: get nginx-path: %s", err)
	}

	err = application.Load(nginxPath, opts.SdsCreds, opts.SdsValidation)
	if err != nil {
		log.Fatalf("envoy-nginx application: load: %s", err)
	}
}
