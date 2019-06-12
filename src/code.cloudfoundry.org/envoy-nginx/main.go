/* Faker envoy.exe */
package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/envoy-nginx/parser"
	fsnotify "github.com/fsnotify/fsnotify"
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

	// locate sds file
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
	if err = parser.GenerateConf(sdsFile, outputDirectory); err != nil {
		log.Fatal(err)
	}

	nginxConf := filepath.Join(outputDirectory, "envoy_nginx.conf")

	// watch sds file
	go watchFile(sdsFile, func() error {
		//TODO Replace cert.pem and key.pem with their new values
		c := exec.Command(nginxBin, "-s", "reload")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err = c.Start(); err != nil {
			log.Fatal(err)
		}
		return err
	})

	// invoke nginx.exe
	c := exec.Command(nginxBin, "-c", nginxConf, "-p", outputDirectory)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err = c.Run(); err != nil {
		log.Fatal(err)
	}
}

func watchFile(filepath string, callback func() error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				// TODO: do we want to return?
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					err := callback()
					if err != nil {
						log.Fatal(err)
					}
				}
			case err, ok := <-watcher.Errors:
				// TODO: do we want to return?
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(filepath)
	if err != nil {
		log.Fatal(err)
	}

	<-done
}
