/* Faker envoy.exe */
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

const sds_path = "C:\\etc\\cf-assets\\envoy_config\\sds-server-cert-and-key.yaml"

var tmpdir string = os.TempDir()

/*
* Try to use this auth_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"?
 */
type sds struct {
	Resources []Resource `yaml:"resources,omitempty"`
}
type Resource struct {
	TLSCertificate TLSCertificate `yaml:"tls_certificate,omitempty"`
}
type TLSCertificate struct {
	CChain     CChain     `yaml:"certificate_chain,omitempty"`
	PrivateKey PrivateKey `yaml:"private_key,omitempty"`
}

type CChain struct {
	InlineString string `yaml:"inline_string,omitempty"`
}

type PrivateKey struct {
	InlineString string `yaml:"inline_string,omitempty"`
}

/* you can do better */
func convertToUnixPath(f string) string {
	f = strings.Replace(f, "C:", "", -1)
	f = strings.Replace(f, "\\", "/", -1)
	return f
}

//TODO: Merge getCert() and getKey()
func getCert() string {
	contents, err := ioutil.ReadFile(sds_path)
	if err != nil {
		panic(err)
	}

	auth := sds{}

	if err := yaml.Unmarshal(contents, &auth); err != nil {
		panic(err)
	}
	return auth.Resources[0].TLSCertificate.CChain.InlineString
}

func getKey() string {
	contents, err := ioutil.ReadFile(sds_path)
	if err != nil {
		panic(err)
	}

	auth := sds{}

	if err := yaml.Unmarshal(contents, &auth); err != nil {
		panic(err)
	}
	return auth.Resources[0].TLSCertificate.PrivateKey.InlineString
}

/* generates conf file and returns it full file path.
* There's aleady a nginx.conf in the blob but that's just a placeholder.
 */

// later TODO: read port mapping from envoy.config
func generateConf() (string, error) {
	certfile := filepath.Join(tmpdir, "cert.pem")
	keyfile := filepath.Join(tmpdir, "key.pem")
	pidfile := filepath.Join(tmpdir, "nginx.pid")
	str := fmt.Sprintf(`
worker_processes  1;
daemon off;

error_log stderr;
pid %s;

events {
    worker_connections  1024;
}


stream {

    upstream app {
      server 127.0.0.1:8080;
    }

    upstream sshd {
      server 127.0.0.1:2222;
    }

    server {
        listen 61001 ssl;
        ssl_certificate      %s;
        ssl_certificate_key  %s;
				proxy_pass app;
    }

    server {
        listen 61002 ssl;
        ssl_certificate      %s;
        ssl_certificate_key  %s;
				proxy_pass sshd;
    }
}
`, convertToUnixPath(pidfile),
		convertToUnixPath(certfile),
		convertToUnixPath(keyfile),
		convertToUnixPath(certfile),
		convertToUnixPath(keyfile))
	cert := getCert()
	key := getKey()

	err := ioutil.WriteFile(certfile, []byte(cert), 0644)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(keyfile, []byte(key), 0644)
	if err != nil {
		panic(err)
	}

	d1 := []byte(str)
	conf := filepath.Join(tmpdir, "envoy_nginx.conf")
	err = ioutil.WriteFile(conf, d1, 0644)
	if err != nil {
		panic(err)
	}
	return conf, nil
}

func main() {
	mypath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	pwd := filepath.Dir(mypath)

	nginx_bin := filepath.Join(pwd, "nginx.exe")
	nginx_conf, e := generateConf()
	if e != nil {
		panic(err)
	}
	c := exec.Command(nginx_bin, "-c", nginx_conf, "-p", tmpdir)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err = c.Run()
	if err != nil {
		log.Fatal(err)
	}
}
