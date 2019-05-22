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
	CertChain  CertChain  `yaml:"certificate_chain,omitempty"`
	PrivateKey PrivateKey `yaml:"private_key,omitempty"`
}

type CertChain struct {
	InlineString string `yaml:"inline_string,omitempty"`
}

type PrivateKey struct {
	InlineString string `yaml:"inline_string,omitempty"`
}

/* Convert windows paths to unix paths */
func convertToUnixPath(path string) string {
	path = strings.Replace(path, "C:", "", -1)
	path = strings.Replace(path, "\\", "/", -1)
	return path
}

/* Parses the Envoy SDS file and extracts the cert and key */
func getCertAndKey(certsFile string) (cert, key string, err error) {
	contents, err := ioutil.ReadFile(certsFile)
	if err != nil {
		return "", "", err
	}

	auth := sds{}

	if err := yaml.Unmarshal(contents, &auth); err != nil {
		return "", "", err
	}

	cert = auth.Resources[0].TLSCertificate.CertChain.InlineString
	key = auth.Resources[0].TLSCertificate.PrivateKey.InlineString
	return cert, key, nil
}

/* Generates NGINX config file and returns its full file path.
 *  There's aleady an nginx.conf in the blob but it's just a placeholder.
 */

// later TODO: read port mapping from envoy.yaml
func generateConf() (string, error) {
	certFile := filepath.Join(tmpdir, "cert.pem")
	keyFile := filepath.Join(tmpdir, "key.pem")
	pidFile := filepath.Join(tmpdir, "nginx.pid")
	confTemplate := fmt.Sprintf(`
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
`, convertToUnixPath(pidFile),
		convertToUnixPath(certFile),
		convertToUnixPath(keyFile),
		convertToUnixPath(certFile),
		convertToUnixPath(keyFile))

	certsFile := "C:\\etc\\cf-assets\\envoy_config\\sds-server-cert-and-key.yaml"
	cert, key, err := getCertAndKey(certsFile)
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(certFile, []byte(cert), 0644)
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(keyFile, []byte(key), 0644)
	if err != nil {
		return "", err
	}

	confFile := filepath.Join(tmpdir, "envoy_nginx.conf")
	err = ioutil.WriteFile(confFile, []byte(confTemplate), 0644)
	if err != nil {
		return "", err
	}
	return confFile, nil
}

func main() {
	mypath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	pwd := filepath.Dir(mypath)

	nginxBin := filepath.Join(pwd, "nginx.exe")
	nginxConf, err := generateConf()
	if err != nil {
		log.Fatal(err)
	}
	c := exec.Command(nginxBin, "-c", nginxConf, "-p", tmpdir)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err = c.Run()
	if err != nil {
		log.Fatal(err)
	}
}
