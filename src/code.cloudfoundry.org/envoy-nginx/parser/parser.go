/* Faker envoy.exe */
package parser

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

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
func getCertAndKey(sdsFile string) (cert, key string, err error) {
	contents, err := ioutil.ReadFile(sdsFile)
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

/* Generates NGINX config file.
 *  There's aleady an nginx.conf in the blob but it's just a placeholder.
 */
func GenerateConf(sdsFile, outputDirectory string) error {
	confFile := filepath.Join(outputDirectory, "envoy_nginx.conf")
	certFile := filepath.Join(outputDirectory, "cert.pem")
	keyFile := filepath.Join(outputDirectory, "key.pem")
	pidFile := filepath.Join(outputDirectory, "nginx.pid")

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

	cert, key, err := getCertAndKey(sdsFile)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(confFile, []byte(confTemplate), 0644)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(certFile, []byte(cert), 0644)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(keyFile, []byte(key), 0644)
	return err
}
