package parser

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"
)

const FilePerm = 0644

type BaseTemplate struct {
	UpstreamAddress, UpstreamPort, ListenerPort, Name, Key, Cert string
}

type envoyConfParser interface {
	GetClusters(file string) ([]Cluster, map[string]string, error)
}

type sdsCredParser interface {
	GetCertAndKey(sdsFile string) (string, string, error)
}

type NginxConfig struct {
	envoyConfParser envoyConfParser
	sdsCredParser   sdsCredParser
	confDir         string
	confFile        string
	certFile        string
	keyFile         string
	pidFile         string
}

func NewNginxConfig(envoyConfParser envoyConfParser, sdsCredParser sdsCredParser, confDir string) NginxConfig {
	return NginxConfig{
		envoyConfParser: envoyConfParser,
		sdsCredParser:   sdsCredParser,
		confDir:         confDir,
		confFile:        filepath.Join(confDir, "envoy_nginx.conf"),
		certFile:        filepath.Join(confDir, "cert.pem"),
		keyFile:         filepath.Join(confDir, "key.pem"),
		pidFile:         filepath.Join(confDir, "nginx.pid"),
	}
}

func (n NginxConfig) GetConfDir() string {
	return n.confDir
}

func (n NginxConfig) GetConfFile() string {
	return n.confFile
}

/* Convert windows paths to unix paths */
func convertToUnixPath(path string) string {
	path = strings.Replace(path, "C:", "", -1)
	path = strings.Replace(path, "\\", "/", -1)
	return path
}

/* Generates NGINX config file.
 *  There's aleady an nginx.conf in the blob but it's just a placeholder.
 */
func (n NginxConfig) Generate(envoyConfFile, sdsFile string) (string, error) {
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
        ssl_certificate      {{.Cert}};
        ssl_certificate_key  {{.Key}};
        proxy_pass {{.Name}};
    }
	`
	//create buffer to store template output
	out := &bytes.Buffer{}

	//Create a new template and parse the conf template into it
	t := template.Must(template.New("baseTemplate").Parse(baseTemplate))

	unixCert := convertToUnixPath(n.certFile)
	unixKey := convertToUnixPath(n.keyFile)

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

error_log stderr;
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

	err = n.WriteCertAndKey(sdsFile)
	if err != nil {
		return "", err
	}

	return n.confFile, nil
}

func (n NginxConfig) WriteCertAndKey(sdsFile string) error {
	cert, key, err := n.sdsCredParser.GetCertAndKey(sdsFile)
	if err != nil {
		return fmt.Errorf("Failed to get cert and key from sds file: %s", err)
	}

	err = ioutil.WriteFile(n.certFile, []byte(cert), FilePerm)
	if err != nil {
		return fmt.Errorf("Failed to write cert file: %s", err)
	}

	err = ioutil.WriteFile(n.keyFile, []byte(key), FilePerm)
	if err != nil {
		return fmt.Errorf("Failed to write key file: %s", err)
	}

	return nil
}
