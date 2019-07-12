package parser

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const FilePerm = 0644

type BaseTemplate struct {
	UpstreamAddress, UpstreamPort, ListenerPort, Name, Key, Cert, TrustedCA string
	TLS                                                                     bool
}

type envoyConfParser interface {
	GetClusters(file string) ([]Cluster, map[string]string, error)
}

type sdsCredParser interface {
	GetCertAndKey() (string, string, error)
}

type sdsValidationParser interface {
	GetCACert() (string, error)
}

type NginxConfig struct {
	envoyConfParser     envoyConfParser
	sdsCredParser       sdsCredParser
	sdsValidationParser sdsValidationParser
	confDir             string
	confFile            string
	certFile            string
	keyFile             string
	trustedCAFile       string
	pidFile             string
}

func NewNginxConfig(envoyConfParser envoyConfParser, sdsCredParser sdsCredParser, sdsValidationParser sdsValidationParser, confDir string) NginxConfig {
	return NginxConfig{
		envoyConfParser:     envoyConfParser,
		sdsCredParser:       sdsCredParser,
		sdsValidationParser: sdsValidationParser,
		confDir:             confDir,
		confFile:            filepath.Join(confDir, "envoy_nginx.conf"),
		certFile:            filepath.Join(confDir, "cert.pem"),
		keyFile:             filepath.Join(confDir, "key.pem"),
		trustedCAFile:       filepath.Join(confDir, "ca.pem"),
		pidFile:             filepath.Join(confDir, "nginx.pid"),
	}
}

func (n NginxConfig) GetConfDir() string {
	return n.confDir
}

func (n NginxConfig) GetConfFile() string {
	return n.confFile
}

// Convert windows paths to unix paths
func convertToUnixPath(path string) string {
	path = strings.Replace(path, "C:", "", -1)
	path = strings.Replace(path, "\\", "/", -1)
	return path
}

// Generates NGINX config file.
func (n NginxConfig) Generate(envoyConfFile string) (string, error) {
	clusters, nameToPortMap, err := n.envoyConfParser.GetClusters(envoyConfFile)
	if err != nil {
		return "", err
	}

	const baseTemplate = `
    upstream {{.Name}} {
      server {{.UpstreamAddress}}:{{.UpstreamPort}};
    }

    server {
        listen {{.ListenerPort}} ssl;
        ssl_certificate        {{.Cert}};
        ssl_certificate_key    {{.Key}};
        {{ if .TLS }}
        ssl_client_certificate {{.TrustedCA}};
        ssl_verify_client on;
        {{ end }}
        proxy_pass {{.Name}};
    }
	`
	//create buffer to store template output
	out := &bytes.Buffer{}

	//Create a new template and parse the conf template into it
	t := template.Must(template.New("baseTemplate").Parse(baseTemplate))

	unixCert := convertToUnixPath(n.certFile)
	unixKey := convertToUnixPath(n.keyFile)
	unixCA := convertToUnixPath(n.trustedCAFile)

	// If there is no trusted CA file, disable tls.
	tls := true
	if _, err := os.Stat(n.trustedCAFile); os.IsNotExist(err) {
		tls = false
	}

	//Execute the template for each socket address
	for _, c := range clusters {
		listenerPort, exists := nameToPortMap[c.Name]
		if !exists {
			return "", fmt.Errorf("port is missing for cluster name %s", c.Name)
		}

		bts := BaseTemplate{
			Name:            c.Name,
			UpstreamAddress: c.Hosts[0].SocketAddress.Address,
			UpstreamPort:    c.Hosts[0].SocketAddress.PortValue,
			Cert:            unixCert,
			Key:             unixKey,
			TrustedCA:       unixCA,
			TLS:             tls,
			ListenerPort:    listenerPort,
		}

		err = t.Execute(out, bts)
		if err != nil {
			return "", fmt.Errorf("Failed executing nginx config template: %s", err)
		}
	}

	confTemplate := fmt.Sprintf(`
worker_processes  1;
daemon on;

error_log logs/error.log;
pid %s;

events {
    worker_connections  1024;
}

stream {
	%s
}
`, convertToUnixPath(n.pidFile),
		out)

	err = ioutil.WriteFile(n.confFile, []byte(confTemplate), FilePerm)
	if err != nil {
		return "", fmt.Errorf("Failed to write envoy_nginx.conf: %s", err)
	}

	return n.confFile, nil
}

func (n NginxConfig) WriteTLSFiles() error {
	cert, key, err := n.sdsCredParser.GetCertAndKey()
	if err != nil {
		return fmt.Errorf("get cert and key from sds server cred parser: %s", err)
	}

	err = ioutil.WriteFile(n.certFile, []byte(cert), FilePerm)
	if err != nil {
		return fmt.Errorf("write cert: %s", err)
	}

	err = ioutil.WriteFile(n.keyFile, []byte(key), FilePerm)
	if err != nil {
		return fmt.Errorf("write key: %s", err)
	}

	caCert, err := n.sdsValidationParser.GetCACert()
	if err != nil {
		return fmt.Errorf("get ca cert from sds server validation parser: %s", err)
	}

	// If there is no CA Cert, do not write the ca.pem.
	if len(caCert) == 0 {
		return nil
	}

	err = ioutil.WriteFile(n.trustedCAFile, []byte(caCert), FilePerm)
	if err != nil {
		return fmt.Errorf("write ca cert file: %s", err)
	}

	return nil
}
