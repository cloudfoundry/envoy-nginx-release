package main

import (
	"io/ioutil"
	"log"
	"os"

	"code.cloudfoundry.org/envoy-nginx/app"
)

func main() {
	flags := app.NewFlags()
	opts := flags.Parse(os.Args[1:])

	stderr := app.NewLogger(os.Stderr)
	tailer := app.NewLogTailer(stderr)

	cmd := app.NewCmd(os.Stdout, os.Stderr)
	stdout := app.NewLogger(os.Stdout)
	application := app.NewApp(stdout, cmd, tailer, opts.EnvoyConfig)

	nginxBinPath, err := application.GetNginxPath()
	if err != nil {
		log.Fatalf("envoy-nginx application: get nginx-path: %s", err)
	}

	nginxConfDir, err := ioutil.TempDir("", "nginx")
	if err != nil {
		log.Fatalf("envoy-nginx application: create nginx config dir: %s", err)
	}

	err = application.Run(nginxConfDir, nginxBinPath, opts.SdsIdCreds, opts.SdsC2CCreds, opts.SdsIdValidation)
	if err != nil {
		log.Fatalf("envoy-nginx application: load: %s", err)
	}
}
