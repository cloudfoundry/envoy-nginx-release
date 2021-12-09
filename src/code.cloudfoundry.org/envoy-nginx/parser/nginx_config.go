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
	Name            string
	UpstreamAddress string
	UpstreamPort    string
	TrustedCA       string
	Servers         []TemplateServer
}

type TemplateServer struct {
	Port    string
	MTLS    bool
	Key     string
	Cert    string
	Ciphers string
}

type envoyConfParser interface {
	ReadUnmarshalEnvoyConfig(envoyConfFile string) (EnvoyConf, error)
	GetClusters(conf EnvoyConf) ([]Cluster, map[string][]ListenerInfo)
}

type SdsCredParser interface {
	GetCertAndKey() (string, string, error)
	ConfigType() SdsConfigType
}

type SdsValidationParser interface {
	GetCACert() (string, error)
}

type NginxConfig struct {
	envoyConfParser     envoyConfParser
	sdsCredParsers      []SdsCredParser
	sdsValidationParser SdsValidationParser
	nginxDir            string
	confFile            string
	idCertFile          string
	idKeyFile           string
	c2cCertFile         string
	c2cKeyFile          string
	trustedCAFile       string
	pidFile             string
}

func NewNginxConfig(envoyConfParser envoyConfParser, sdsCredParsers []SdsCredParser, sdsValidationParser SdsValidationParser, nginxDir string) NginxConfig {
	return NginxConfig{
		envoyConfParser:     envoyConfParser,
		sdsCredParsers:      sdsCredParsers,
		sdsValidationParser: sdsValidationParser,
		nginxDir:            nginxDir,
		confFile:            filepath.Join(nginxDir, "conf", "nginx.conf"),
		idCertFile:          filepath.Join(nginxDir, "id-cert.pem"),
		idKeyFile:           filepath.Join(nginxDir, "id-key.pem"),
		trustedCAFile:       filepath.Join(nginxDir, "id-ca.pem"),
		c2cCertFile:         filepath.Join(nginxDir, "c2c-cert.pem"),
		c2cKeyFile:          filepath.Join(nginxDir, "c2c-key.pem"),
		pidFile:             filepath.Join(nginxDir, "nginx.pid"),
	}
}

func (n NginxConfig) GetNginxDir() string {
	return n.nginxDir
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
func (n NginxConfig) Generate(envoyConfFile string) error {
	envoyConf, err := n.envoyConfParser.ReadUnmarshalEnvoyConfig(envoyConfFile)
	if err != nil {
		return fmt.Errorf("read and unmarshal Envoy config: %s", err)
	}

	clusters, nameToListeners := n.envoyConfParser.GetClusters(envoyConf)

	const baseTemplate = `
    upstream {{.Name}} {
      server {{.UpstreamAddress}}:{{.UpstreamPort}};
    }

    {{range .Servers}}
    server {
        listen {{.Port}} ssl;
        ssl_certificate        {{.Cert}};
        ssl_certificate_key    {{.Key}};
        {{ if .MTLS }}
        ssl_client_certificate {{$.TrustedCA}};
        ssl_verify_client on;
        {{ end }}
        proxy_pass {{$.Name}};

				ssl_prefer_server_ciphers on;
				ssl_ciphers {{.Ciphers}};
    }
	{{end}}
	`
	//create buffer to store template output
	out := &bytes.Buffer{}

	//Create a new template and parse the conf template into it
	t := template.Must(template.New("baseTemplate").Parse(baseTemplate))

	unixIdCert := convertToUnixPath(n.idCertFile)
	unixIdKey := convertToUnixPath(n.idKeyFile)
	unixC2CCert := convertToUnixPath(n.c2cCertFile)
	unixC2CKey := convertToUnixPath(n.c2cKeyFile)
	unixCA := convertToUnixPath(n.trustedCAFile)

	//Execute the template for each socket address
	for _, c := range clusters {
		bts := BaseTemplate{
			Name:            c.Name,
			UpstreamAddress: c.LoadAssignment.Endpoints[0].LBEndpoints[0].Endpoint.Address.SocketAddress.Address,
			UpstreamPort:    c.LoadAssignment.Endpoints[0].LBEndpoints[0].Endpoint.Address.SocketAddress.PortValue,
			TrustedCA:       unixCA,
		}

		for _, listener := range nameToListeners[c.Name] {
			var cert, key string
			if listener.SdsConfigType == SdsIdConfigType {
				cert = unixIdCert
				key = unixIdKey
			} else {
				cert = unixC2CCert
				key = unixC2CKey
			}
			bts.Servers = append(bts.Servers, TemplateServer{
				Port:    listener.Port,
				MTLS:    listener.MTLS,
				Cert:    cert,
				Key:     key,
				Ciphers: listener.Ciphers,
			})
		}

		err = t.Execute(out, bts)
		if err != nil {
			return fmt.Errorf("executing envoy-nginx config template: %s", err)
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
		return fmt.Errorf("%s - write file failed: %s", n.confFile, err)
	}

	return nil
}

func (n NginxConfig) WriteTLSFiles() error {
	for _, sdsCredParser := range n.sdsCredParsers {
		cert, key, err := sdsCredParser.GetCertAndKey()
		if err != nil {
			return fmt.Errorf("get cert and key from sds cred parser: %s", err)
		}

		var certFile, keyFile string
		if sdsCredParser.ConfigType() == SdsIdConfigType {
			certFile = n.idCertFile
			keyFile = n.idKeyFile
		} else {
			certFile = n.c2cCertFile
			keyFile = n.c2cKeyFile
		}

		err = ioutil.WriteFile(certFile, []byte(cert), FilePerm)
		if err != nil {
			return fmt.Errorf("write cert: %s", err)
		}

		err = ioutil.WriteFile(keyFile, []byte(key), FilePerm)
		if err != nil {
			return fmt.Errorf("write key: %s", err)
		}
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
