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
	cmd         cmd
}

type logger interface {
	Println(string)
}

type cmd interface {
	Run(string, ...string) error
}

func NewApp(logger logger, cmd cmd, envoyConfig string) App {
	return App{
		envoyConfig: envoyConfig,
		logger:      logger,
		cmd:         cmd,
	}
}

// Searching for nginx.exe in the same directory
// that our app binary is running in.
func (a App) GetNginxPath() (path string, err error) {
	log.SetOutput(os.Stdout)

	mypath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("executable path: %s", err)
	}

	pwd := filepath.Dir(mypath)
	nginxPath := filepath.Join(pwd, "nginx.exe")

	_, err = os.Stat(nginxPath)
	if err != nil {
		return "", fmt.Errorf("stat nginx.exe: %s", err)
	}

	return nginxPath, nil
}

// Setting up nginx config and directory.
// Creating two goroutines: one to watch the sds creds file
// and reload nginx when the creds rotate, the other to start
// nginx.
func (a App) Load(nginxPath, sdsCreds, sdsValidation string) error {
	log.SetOutput(os.Stdout)

	confDir, err := ioutil.TempDir("", "nginx-conf")
	if err != nil {
		return fmt.Errorf("create nginx-conf dir: %s", err)
	}

	/*
	* The only reason to do this to suppress the nginx error complaining about
	* missing "logs/error.log". This is because the nginx.exe in the blob
	* was compiled this location wired into it.
	 */
	err = os.Mkdir(filepath.Join(confDir, "logs"), os.ModePerm)
	if err != nil {
		return fmt.Errorf("create nginx-conf/logs dir: %s", err)
	}

	envoyConfParser := parser.NewEnvoyConfParser()
	sdsCredParser := parser.NewSdsCredParser(sdsCreds)
	sdsValidationParser := parser.NewSdsServerValidationParser(sdsValidation)

	nginxConfParser := parser.NewNginxConfig(envoyConfParser, sdsCredParser, sdsValidationParser, confDir)

	errorChan := make(chan error)
	readyChan := make(chan bool)

	go func() {
		errorChan <- WatchFile(sdsCreds, readyChan, func() error {
			log.Printf("detected change in sdsfile (%s)\n", sdsCreds)
			sdsFd, err := os.Stat(sdsCreds)
			if err != nil {
				return fmt.Errorf("stat sds-server-cert-and-key.yaml: %s", err)
			}
			/* It's observed that sometimes fsnotify may provide a double notification
			* with one of the notifications reporting an empty file. NOOP in that case
			 */
			if sdsFd.Size() < 1 {
				log.Printf("detected change in sdsfile (%s) was a false alarm. NOOP.\n", sdsCreds)
				return nil
			}
			return reloadNginx(nginxPath, nginxConfParser)
		})
	}()

	go func() {
		<-readyChan
		errorChan <- a.startNginx(nginxPath, nginxConfParser, a.envoyConfig)
	}()

	err = <-errorChan
	if err != nil {
		return err
	}

	return nil
}

// Rotates cert, key, and ca cert in nginx config directory.
// Reloads nginx.
func reloadNginx(nginxPath string, nginxConfParser parser.NginxConfig) error {
	log.Println("envoy.exe: about to reload nginx")

	err := nginxConfParser.WriteTLSFiles()
	if err != nil {
		return fmt.Errorf("Failed to write tls files: %s", err)
	}

	confDir := nginxConfParser.GetConfDir()
	confFile := nginxConfParser.GetConfFile()

	log.Println("envoy.exe: about to issue -s reload")
	log.Println("envoy.exe: Executing:", nginxPath, "-c", confFile, "-p", confDir, "-s", "reload")

	c := exec.Command(nginxPath, "-c", confFile, "-p", confDir, "-s", "reload")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// Generates nginx config from envoy config.
// Writes cert, key, and ca cert to files in nginx config directory.
// Starts nginx.
func (a App) startNginx(nginxPath string, nginxConfParser parser.NginxConfig, envoyConf string) error {
	confFile, err := nginxConfParser.Generate(envoyConf)
	if err != nil {
		return fmt.Errorf("generate nginx config from envoy config: %s", err)
	}

	err = nginxConfParser.WriteTLSFiles()
	if err != nil {
		return fmt.Errorf("write tls files: %s", err)
	}

	confDir := nginxConfParser.GetConfDir()

	a.logger.Println(fmt.Sprintf("start nginx: %s -c %s -p %s", nginxPath, confFile, confDir))

	err = a.cmd.Run(nginxPath, "-c", confFile, "-p", confDir)
	if err != nil {
		return fmt.Errorf("cmd run: %s", err)
	}
	return nil
}
