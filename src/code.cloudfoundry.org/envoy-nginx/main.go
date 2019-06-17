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

// TODO: read from -c. see executor/transformer.go
const DefaultEnvoyConfFile = "C:\\etc\\cf-assets\\envoy_config\\envoy.yaml"
const DefaultSDSCredsFile = "C:\\etc\\cf-assets\\envoy_config\\sds-server-cert-and-key.yaml"

func main() {
	log.Println("envoy.exe: Starting executable")
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

	outputDirectory, err := ioutil.TempDir("", "nginx-conf")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("envoy.exe: Generating conf")
	// generate config
	envoyConfParser := parser.NewEnvoyConfParser()
	sdsCredParser := parser.NewSdsCredParser()
	p := parser.NewParser(envoyConfParser, sdsCredParser)
	err = p.GenerateConf(envoyConfFile, sdsFile, outputDirectory)
	if err != nil {
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
			log.Printf("envoy.exe: detected change in sdsfile (%s)\n", sdsFile)
			sdsFd, err := os.Stat(sdsFile)
			if err != nil {
				return err
			}
			/* It's observed that sometimes fsnotify may provide a double notification
			* with one of the notifications reporting an empty file. NOOP in that case
			 */
			if sdsFd.Size() < 1 {
				log.Printf("envoy.exe: detected change in sdsfile (%s) was a false alarm. NOOP.\n", sdsFile)
				return nil
			}
			return reloadNginx(nginxBin, nginxConf, sdsFile, outputDirectory, envoyConfFile, p)
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

func reloadNginx(nginxBin, nginxConf, sdsFile, outputDirectory, envoyConfFile string, p parser.Parser) error {
	log.Println("envoy.exe: about to reload nginx")
	if err := p.GenerateConf(envoyConfFile, sdsFile, outputDirectory); err != nil {
		return err
	}

	/*
	* The reason we need to be explicit about the the -c and -p is because the nginx.exe
	* we use (as of date) is wired during compilation to use "./conf/nginx.conf" as
	 */
	log.Println("envoy.exe: about to issue -s reload")
	log.Println("envoy.exe: Executing:", nginxBin, "-c", nginxConf, "-p", outputDirectory, "-s", "reload")
	c := exec.Command(nginxBin, "-c", nginxConf, "-p", outputDirectory, "-s", "reload")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func executeNginx(nginxBin, nginxConf, outputDirectory string) error {
	log.Println("envoy.exe: Executing:", nginxBin, "-c", nginxConf, "-p", outputDirectory)
	c := exec.Command(nginxBin, "-c", nginxConf, "-p", outputDirectory)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
