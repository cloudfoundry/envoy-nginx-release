/* Faker envoy.exe */
package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	parser "code.cloudfoundry.org/envoy-nginx/parser"
)

func main() {
	mypath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	pwd := filepath.Dir(mypath)

	nginxBin := filepath.Join(pwd, "nginx.exe")
	nginxConf, err := parser.GenerateConf()
	if err != nil {
		log.Fatal(err)
	}
	var tmpdir string = os.TempDir()
	c := exec.Command(nginxBin, "-c", nginxConf, "-p", tmpdir)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err = c.Run()
	if err != nil {
		log.Fatal(err)
	}
}
