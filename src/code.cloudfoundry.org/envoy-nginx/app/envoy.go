package app

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/envoy-nginx/parser"
)

type App struct {
	envoyConfig string
	logger      logger
}
type logger interface {
	Println(string)
}

func NewApp(logger logger, envoyConfig string) App {
	return App{
		envoyConfig: envoyConfig,
		logger:      logger,
	}
}

func (a App) Load(sdsCreds, sdsValidation string) error {
	log.SetOutput(os.Stdout)
	log.Println("finding nginx executable")
	// locate nginx.exe in the same directory as the running executable
	mypath, err := os.Executable()
	if err != nil {
		// TODO: Format & test error
		return err
	}

	pwd := filepath.Dir(mypath)
	nginxBin := filepath.Join(pwd, "nginx.exe")

	if _, err = os.Stat(nginxBin); err != nil {
		return fmt.Errorf("os stat nginx.exe: %s", err)
	}

	confDir, err := ioutil.TempDir("", "nginx-conf")
	if err != nil {
		// TODO: Format & test error
		return err
	}

	log.Println("generating nginx config")
	envoyConfParser := parser.NewEnvoyConfParser()
	sdsCredParser := parser.NewSdsCredParser(sdsCreds)
	sdsValidationParser := parser.NewSdsServerValidationParser(sdsValidation)

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
		errorChan <- WatchFile(sdsCreds, readyChan, func() error {
			log.Printf("detected change in sdsfile (%s)\n", sdsCreds)
			sdsFd, err := os.Stat(sdsCreds)
			if err != nil {
				// TODO: Format & test error
				return err
			}
			/* It's observed that sometimes fsnotify may provide a double notification
			* with one of the notifications reporting an empty file. NOOP in that case
			 */
			if sdsFd.Size() < 1 {
				log.Printf("detected change in sdsfile (%s) was a false alarm. NOOP.\n", sdsCreds)
				return nil
			}
			return reloadNginx(nginxBin, nginxConfParser)
		})
	}()

	go func() {
		<-readyChan
		errorChan <- startNginx(nginxBin, nginxConfParser, a.envoyConfig)
	}()

	err = <-errorChan
	if err != nil {
		// TODO: Format & test error
		return err
	}

	return nil
}

func reloadNginx(nginxBin string, nginxConfParser parser.NginxConfig) error {
	log.Println("envoy.exe: about to reload nginx")

	err := nginxConfParser.WriteTLSFiles()
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

func startNginx(nginxBin string, nginxConfParser parser.NginxConfig, envoyConf string) error {
	confFile, err := nginxConfParser.Generate(envoyConf)
	if err != nil {
		return fmt.Errorf("Generating nginx config from envoy config: %s", err)
	}

	err = nginxConfParser.WriteTLSFiles()
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
