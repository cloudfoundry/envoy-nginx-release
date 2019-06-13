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

	sdsFile := os.Getenv("SDS_FILE")
	if sdsFile == "" {
		sdsFile = DefaultSDSCredsFile
	}

	outputDirectory, err := ioutil.TempDir("", "nginx-conf")
	if err != nil {
		log.Fatal(err)
	}

	if err = parser.GenerateConf(sdsFile, outputDirectory); err != nil {
		log.Fatal(err)
	}

	nginxConf := filepath.Join(outputDirectory, "envoy_nginx.conf")

	/*
	* The idea here is that the main line of execution waits for any errors
	* on the @errorChan.
	* There are 2 go funcs spun out - (1) executing nginx and (2) watching the SDS file.
	* They publish errors (if any) to this error channel
	 */
	errorChan := make(chan error)

	go func() {
		errorChan <- watchFile(sdsFile, func() error {
			// TODO: we observed during our testing that fsnotify
			// can trigger "twice" when the file is re-written,
			// and once we saw that the file contents were empty,
			// causing the parser to fail.
			// We should consider returning nil from the callback
			// if the sds file is empty.
			return reloadNginx(nginxBin, sdsFile, outputDirectory)
		})
	}()

	go func() {
		errorChan <- executeNginx(nginxBin, nginxConf, outputDirectory)
	}()

	err = <-errorChan
	if err != nil {
		log.Fatal(err)
	}
}

func reloadNginx(nginxBin, sdsFile, outputDirectory string) error {
	var err error
	if err = parser.GenerateConf(sdsFile, outputDirectory); err != nil {
		return err
	}
	c := exec.Command(nginxBin, "-s", "reload")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func executeNginx(nginxBin, nginxConf, outputDirectory string) error {
	c := exec.Command(nginxBin, "-c", nginxConf, "-p", outputDirectory)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
