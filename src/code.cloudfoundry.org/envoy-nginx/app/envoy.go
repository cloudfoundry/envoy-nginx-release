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
	logger      logger
	cmd         cmd
	tailer      tailer
	envoyConfig string
}

type logger interface {
	Println(string)
}

type tailer interface {
	Tail(string) error
}

type cmd interface {
	Run(string, ...string) error
}

func NewApp(logger logger, cmd cmd, tailer tailer, envoyConfig string) App {
	return App{
		logger:      logger,
		cmd:         cmd,
		tailer:      tailer,
		envoyConfig: envoyConfig,
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

	err = os.Mkdir(filepath.Join(confDir, "logs"), os.ModePerm)
	if err != nil {
		return fmt.Errorf("create nginx-conf/logs dir for error.log: %s", err)
	}

	err = os.Mkdir(filepath.Join(confDir, "conf"), os.ModePerm)
	if err != nil {
		return fmt.Errorf("create nginx-conf/conf dir for nginx.conf: %s", err)
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
	err := nginxConfParser.WriteTLSFiles()
	if err != nil {
		return fmt.Errorf("write tls files: %s", err)
	}

	confDir := nginxConfParser.GetConfDir()

	log.Println("envoy-nginx application: reload nginx:", nginxPath, "-p", confDir, "-s", "reload")

	c := exec.Command(nginxPath, "-p", confDir, "-s", "reload")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// Generates nginx config from envoy config.
// Writes cert, key, and ca cert to files in nginx config directory.
// Starts nginx.
func (a App) startNginx(nginxPath string, nginxConfParser parser.NginxConfig, envoyConf string) error {
	err := nginxConfParser.WriteTLSFiles()
	if err != nil {
		return fmt.Errorf("write tls files: %s", err)
	}

	err = nginxConfParser.Generate(envoyConf)
	if err != nil {
		return fmt.Errorf("generate nginx config from envoy config: %s", err)
	}

	confDir := nginxConfParser.GetConfDir()

	a.logger.Println("envoy-nginx application: tailing error log")
	err = a.tailer.Tail(filepath.Join(confDir, "logs", "error.log"))
	if err != nil {
		return fmt.Errorf("tail error log: %s", err)
	}

	a.logger.Println(fmt.Sprintf("envoy-nginx application: start nginx: %s -p %s", nginxPath, confDir))

	err = a.cmd.Run(nginxPath, "-p", confDir)
	if err != nil {
		return fmt.Errorf("cmd run: %s", err)
	}
	return nil
}
