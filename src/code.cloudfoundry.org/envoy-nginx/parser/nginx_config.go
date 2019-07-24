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
	UpstreamAddress string
	UpstreamPort    string
	ListenerPort    string
	Name            string
	Key             string
	Cert            string
	TrustedCA       string
	MTLS            bool
	Ciphers         string
}

type envoyConfParser interface {
	ReadUnmarshalEnvoyConfig(envoyConfFile string) (EnvoyConf, error)
	GetClusters(conf EnvoyConf) ([]Cluster, map[string]string)
	GetMTLS(conf EnvoyConf) bool
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
	envoyConf, err := n.envoyConfParser.ReadUnmarshalEnvoyConfig(envoyConfFile)
	if err != nil {
		return "", fmt.Errorf("read and unmarshal Envoy config: %s", err)
	}

	clusters, nameToPortMap := n.envoyConfParser.GetClusters(envoyConf)

	const baseTemplate = `
    upstream {{.Name}} {
      server {{.UpstreamAddress}}:{{.UpstreamPort}};
    }

    server {
        listen {{.ListenerPort}} ssl;
        ssl_certificate        {{.Cert}};
        ssl_certificate_key    {{.Key}};
        {{ if .MTLS }}
        ssl_client_certificate {{.TrustedCA}};
        ssl_verify_client on;
        {{ end }}
        proxy_pass {{.Name}};

				ssl_prefer_server_ciphers on;
				ssl_ciphers {{.Ciphers}};
    }
	`
	//create buffer to store template output
	out := &bytes.Buffer{}

	//Create a new template and parse the conf template into it
	t := template.Must(template.New("baseTemplate").Parse(baseTemplate))

	unixCert := convertToUnixPath(n.certFile)
	unixKey := convertToUnixPath(n.keyFile)
	unixCA := convertToUnixPath(n.trustedCAFile)

	mtlsEnabled := n.envoyConfParser.GetMTLS(envoyConf)

	supportedCipherSuites := "ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256"

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
			MTLS:            mtlsEnabled,

			ListenerPort: listenerPort,
			Ciphers:      supportedCipherSuites,
		}

		err = t.Execute(out, bts)
		if err != nil {
			return "", fmt.Errorf("executing envoy-nginx config template: %s", err)
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
		return "", fmt.Errorf("write envoy_nginx.conf: %s", err)
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
