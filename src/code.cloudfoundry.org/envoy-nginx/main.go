/* Faker envoy.exe */
package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/envoy-nginx/parser"
)

const DefaultEnvoyConfFile = "C:\\etc\\cf-assets\\envoy_config\\envoy.yaml"
const DefaultSDSCredsFile = "C:\\etc\\cf-assets\\envoy_config\\sds-server-cert-and-key.yaml"

func main() {
	// locate nginx.exe in the same directory as the running executable
	mypath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	pwd := filepath.Dir(mypath)
	nginxBin := filepath.Join(pwd, "nginx.exe")

	if _, err = os.Stat(nginxBin); err != nil {
		log.Fatal(err)
	}

	// locate envoy file - We use ENVOY_FILE for our tests
	envoyConfFile := os.Getenv("ENVOY_FILE")
	if envoyConfFile == "" {
		envoyConfFile = DefaultEnvoyConfFile
	}

	// locate sds file - We use SDS_FILE for our tests
	sdsFile := os.Getenv("SDS_FILE")
	if sdsFile == "" {
		sdsFile = DefaultSDSCredsFile
	}

	// set output directory to be a temporary directory
	outputDirectory, err := ioutil.TempDir("", "nginx-conf")
	if err != nil {
		log.Fatal(err)
	}

	// generate config
	if err = parser.GenerateConf(envoyConfFile, sdsFile, outputDirectory); err != nil {
		log.Fatal(err)
	}

	nginxConf := filepath.Join(outputDirectory, "envoy_nginx.conf")

	// invoke nginx.exe
	c := exec.Command(nginxBin, "-c", nginxConf, "-p", outputDirectory)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err = c.Run(); err != nil {
		log.Fatal(err)
	}
}
