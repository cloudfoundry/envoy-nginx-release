package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	yaml "gopkg.in/yaml.v2"
)

type BaseTemplate struct {
	UpstreamAddress, UpstreamPort, ListenerPort, Name, Key, Cert string
}

/*
* Try to use this auth_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"?
 */
type Sds struct {
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

type envoyConfParser interface {
	GetClusters(file string) ([]Cluster, map[string]string, error)
}

type Parser struct {
	envoyConfParser envoyConfParser
}

func NewParser(envoyConfParser envoyConfParser) Parser {
	return Parser{envoyConfParser: envoyConfParser}
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
		return "", "", fmt.Errorf("Failed to read sds creds: %s", err)
	}

	auth := Sds{}

	err = yaml.Unmarshal(contents, &auth)
	if err != nil {
		return "", "", fmt.Errorf("Failed to unmarshal sds creds: %s", err)
	}

	if len(auth.Resources) < 1 {
		return "", "", errors.New("resources section not found in sds-server-cert-and-key.yaml")
	}

	cert = auth.Resources[0].TLSCertificate.CertChain.InlineString
	key = auth.Resources[0].TLSCertificate.PrivateKey.InlineString
	return cert, key, nil
}

/* Generates NGINX config file.
 *  There's aleady an nginx.conf in the blob but it's just a placeholder.
 */
func (p Parser) GenerateConf(envoyConfFile, sdsFile, outputDirectory string) error {
	confFile := filepath.Join(outputDirectory, "envoy_nginx.conf")
	certFile := filepath.Join(outputDirectory, "cert.pem")
	keyFile := filepath.Join(outputDirectory, "key.pem")
	pidFile := filepath.Join(outputDirectory, "nginx.pid")

	clusters, nameToPortMap, err := p.envoyConfParser.GetClusters(envoyConfFile)
	if err != nil {
		return err
	}

	const baseTemplate = `
    upstream {{.Name}} {
      server {{.UpstreamAddress}}:{{.UpstreamPort}};
    }

    server {
        listen {{.ListenerPort}} ssl;
        ssl_certificate      {{.Cert}};
        ssl_certificate_key  {{.Key}};
        proxy_pass {{.Name}};
    }
	`
	//create buffer to store template output
	out := &bytes.Buffer{}

	//Create a new template and parse the conf template into it
	t := template.Must(template.New("baseTemplate").Parse(baseTemplate))

	unixCert := convertToUnixPath(certFile)
	unixKey := convertToUnixPath(keyFile)

	//Execute the template for each socket address
	for _, c := range clusters {
		listenerPort, exists := nameToPortMap[c.Name]
		if !exists {
			return fmt.Errorf("port is missing for cluster name %s", c.Name)
		}

		bts := BaseTemplate{
			Name:            c.Name,
			UpstreamAddress: c.Hosts[0].SocketAddress.Address,
			UpstreamPort:    c.Hosts[0].SocketAddress.PortValue,
			Cert:            unixCert,
			Key:             unixKey,
			ListenerPort:    listenerPort,
		}

		err = t.Execute(out, bts)
		if err != nil {
			// TODO: add a test
			return err
		}
	}

	confTemplate := fmt.Sprintf(`
worker_processes  1;
daemon on;

error_log stderr;
pid %s;

events {
    worker_connections  1024;
}


stream {
	%s
}
`, convertToUnixPath(pidFile),
		out)

	cert, key, err := getCertAndKey(sdsFile)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(confFile, []byte(confTemplate), 0644)
	if err != nil {
		// TODO: Handle error
		panic(err)
	}

	err = ioutil.WriteFile(certFile, []byte(cert), 0644)
	if err != nil {
		// TODO: Handle error
		panic(err)
	}

	err = ioutil.WriteFile(keyFile, []byte(key), 0644)
	if err != nil {
		// TODO: Handle error
		panic(err)
	}

	return nil
}
