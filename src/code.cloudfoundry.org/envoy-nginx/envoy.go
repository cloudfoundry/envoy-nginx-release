package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/envoy-nginx/parser"
)

const DefaultSDSCredsFile = "C:\\etc\\cf-assets\\envoy_config\\sds-server-cert-and-key.yaml"
const DefaultSDSServerValidationContextFile = "C:\\etc\\cf-assets\\envoy_config\\sds-server-validation-context.yaml"

func envoy(envoyConf string) {
	log.SetOutput(os.Stdout)
	log.Println("envoy.exe: Starting executable")
	// locate nginx.exe in the same directory as the running executable
	mypath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	pwd := filepath.Dir(mypath)
	nginxBin := filepath.Join(pwd, "nginx.exe")

	if _, err = os.Stat(nginxBin); err != nil {
		log.Fatalf("Failed to locate nginx.exe: %s", err)
	}

	// We use SDS_FILE for our tests
	// TODO: can we assume that the sds file will always be in the same directory as the envoy.yaml?
	sdsCredsFile := os.Getenv("SDS_CREDS_FILE")
	if sdsCredsFile == "" {
		sdsCredsFile = DefaultSDSCredsFile
	}

	// We use SDS_FILE for our tests
	sdsValidationFile := os.Getenv("SDS_VALIDATION_FILE")
	if sdsValidationFile == "" {
		sdsValidationFile = DefaultSDSServerValidationContextFile
	}

	confDir, err := ioutil.TempDir("", "nginx-conf")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("envoy.exe: Generating conf")
	envoyConfParser := parser.NewEnvoyConfParser()
	sdsCredParser := parser.NewSdsCredParser()
	sdsValidationParser := parser.NewSdsServerValidationParser()
	nginxConfParser := parser.NewNginxConfig(envoyConfParser, sdsCredParser, sdsValidationParser, confDir)

	/*
	* The idea here is that the main line of execution waits for any errors
	* on the @errorChan.
	* There are 2 go funcs spun out - (1) executing nginx and (2) watching the SDS file.
	* They publish errors (if any) to this error channel
	 */
	errorChan := make(chan error)
	readyChan := make(chan bool)

	go func() {
		errorChan <- WatchFile(sdsCredsFile, readyChan, func() error {
			log.Printf("envoy.exe: detected change in sdsfile (%s)\n", sdsCredsFile)
			sdsFd, err := os.Stat(sdsCredsFile)
			if err != nil {
				return err
			}
			/* It's observed that sometimes fsnotify may provide a double notification
			* with one of the notifications reporting an empty file. NOOP in that case
			 */
			if sdsFd.Size() < 1 {
				log.Printf("envoy.exe: detected change in sdsfile (%s) was a false alarm. NOOP.\n", sdsCredsFile)
				return nil
			}
			return reloadNginx(nginxBin, sdsCredsFile, sdsValidationFile, nginxConfParser)
		})
	}()

	go func() {
		<-readyChan
		errorChan <- startNginx(nginxBin, sdsCredsFile, sdsValidationFile, nginxConfParser, envoyConf)
	}()

	err = <-errorChan
	if err != nil {
		log.Fatal(err)
	}
}

func reloadNginx(nginxBin, sdsCredsFile, sdsValidationFile string, nginxConfParser parser.NginxConfig) error {
	log.Println("envoy.exe: about to reload nginx")

	err := nginxConfParser.WriteTLSFiles(sdsCredsFile, sdsValidationFile)
	if err != nil {
		return fmt.Errorf("Failed to write tls files: %s", err)
	}

	confDir := nginxConfParser.GetConfDir()
	confFile := nginxConfParser.GetConfFile()
	/*
	* The reason we need to be explicit about the the -c and -p is because the nginx.exe
	* we use (as of date) is wired during compilation to use "./conf/nginx.conf" as
	 */
	log.Println("envoy.exe: about to issue -s reload")
	log.Println("envoy.exe: Executing:", nginxBin, "-c", confFile, "-p", confDir, "-s", "reload")
	c := exec.Command(nginxBin, "-c", confFile, "-p", confDir, "-s", "reload")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func startNginx(nginxBin, sdsCredsFile, sdsValidationFile string, nginxConfParser parser.NginxConfig, envoyConf string) error {
	confFile, err := nginxConfParser.Generate(envoyConf)
	if err != nil {
		log.Fatal(err)
	}

	err = nginxConfParser.WriteTLSFiles(sdsCredsFile, sdsValidationFile)
	if err != nil {
		return fmt.Errorf("Failed to write tls files: %s", err)
	}

	confDir := nginxConfParser.GetConfDir()

	log.Println("envoy.exe: Executing:", nginxBin, "-c", confFile, "-p", confDir)

	c := exec.Command(nginxBin, "-c", confFile, "-p", confDir)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
