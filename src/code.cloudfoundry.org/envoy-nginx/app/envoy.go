package app

import (
	"fmt"
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
	nginxBin    string
}

type logger interface {
	Println(...interface{})
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
		// Will be set on Run()
		nginxBin: "",
	}
}

func (a *App) SetNginxBin(nginxPath string) {
	a.nginxBin = nginxPath
}

// Searching for nginx.exe in the same directory
// that our app binary is running in.
func (a App) GetNginxPath() (path string, err error) {
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
func (a App) Run(nginxConfDir, nginxBinPath, sdsIdCreds, sdsC2CCreds, sdsIdValidation string) error {
	a.SetNginxBin(nginxBinPath)

	err := os.Mkdir(filepath.Join(nginxConfDir, "logs"), 0755)
	if err != nil {
		return fmt.Errorf("create nginx/logs dir for error.log: %s", err)
	}

	err = os.Mkdir(filepath.Join(nginxConfDir, "conf"), 0755)
	if err != nil {
		return fmt.Errorf("create nginx/conf dir for nginx.conf: %s", err)
	}

	envoyConfParser := parser.NewEnvoyConfParser()

	sdsIdCredParser := parser.NewSdsIdCredParser(sdsIdCreds)
	sdsCredParsers := []parser.SdsCredParser{sdsIdCredParser}
	if sdsC2CCreds != "" {
		sdsC2CCredParser := parser.NewSdsC2CCredParser(sdsC2CCreds)
		sdsCredParsers = append(sdsCredParsers, sdsC2CCredParser)
	}
	sdsIdValidationParser := parser.NewSdsIdValidationParser(sdsIdValidation)

	nginxConfParser := parser.NewNginxConfig(envoyConfParser, sdsCredParsers, sdsIdValidationParser, nginxConfDir)

	errorChan := make(chan error)
	readyChan := make(chan bool)

	go func() {
		errorChan <- WatchFile(sdsIdCreds, readyChan, func() error {
			return a.sdsFileUpdated(sdsIdCreds, nginxConfParser)
		})
	}()

	if sdsC2CCreds != "" {
		go func() {
			errorChan <- WatchFile(sdsC2CCreds, readyChan, func() error {
				return a.sdsFileUpdated(sdsC2CCreds, nginxConfParser)
			})
		}()
	}

	go func() {
		<-readyChan
		errorChan <- a.startNginx(nginxConfParser)
	}()

	err = <-errorChan
	if err != nil {
		return err
	}

	return nil
}

func (a App) sdsFileUpdated(fileName string, nginxConfParser parser.NginxConfig) error {
	a.logger.Println(fmt.Sprintf("detected change in sdsfile: %s \n", fileName))

	sdsFd, err := os.Stat(fileName)
	if err != nil {
		return fmt.Errorf("stat %s: %s", fileName, err)
	}
	/* It's observed that sometimes fsnotify may provide a double notification
	* with one of the notifications reporting an empty file. NOOP in that case
	 */
	if sdsFd.Size() < 1 {
		a.logger.Println("detected change in sdsfile was a false alarm. NOOP.\n")
		return nil
	}
	return a.reloadNginx(nginxConfParser)
}

// Rotates cert, key, and ca cert in nginx config directory.
// Reloads nginx.
func (a App) reloadNginx(nginxConfParser parser.NginxConfig) error {
	err := nginxConfParser.WriteTLSFiles()
	if err != nil {
		return fmt.Errorf("write tls files: %s", err)
	}

	nginxDir := nginxConfParser.GetNginxDir()

	a.logger.Println("envoy-nginx application: reload nginx:", a.nginxBin, "-p", nginxDir, "-s", "reload")

	c := exec.Command(a.nginxBin, "-p", nginxDir, "-s", "reload")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// Generates nginx config from envoy config.
// Writes cert, key, and ca cert to files in nginx config directory.
// Starts nginx.
func (a App) startNginx(nginxConfParser parser.NginxConfig) error {
	err := nginxConfParser.WriteTLSFiles()
	if err != nil {
		return fmt.Errorf("write tls files: %s", err)
	}

	err = nginxConfParser.Generate(a.envoyConfig)
	if err != nil {
		return fmt.Errorf("generate nginx config from envoy config: %s", err)
	}

	nginxDir := nginxConfParser.GetNginxDir()

	a.logger.Println("envoy-nginx application: tailing error log")
	err = a.tailer.Tail(filepath.Join(nginxDir, "logs", "error.log"))
	if err != nil {
		return fmt.Errorf("tail error log: %s", err)
	}

	a.logger.Println(fmt.Sprintf("envoy-nginx application: start nginx: %s -p %s", a.nginxBin, nginxDir))

	err = a.cmd.Run(a.nginxBin, "-p", nginxDir)
	if err != nil {
		return fmt.Errorf("cmd run: %s", err)
	}
	return nil
}
