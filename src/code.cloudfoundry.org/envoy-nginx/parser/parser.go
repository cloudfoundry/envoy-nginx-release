/* Faker envoy.exe */
package parser

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	yaml "gopkg.in/yaml.v2"
)

type EnvoyConf struct {
	StaticResources StaticResources `yaml:"static_resources,omitempty"`
}

type StaticResources struct {
	Clusters  []Cluster  `yaml:"clusters,omitempty"`
	Listeners []Listener `yaml:"listeners,omitempty"`
}

type Cluster struct {
	Hosts []Host `yaml:"hosts,omitempty"`
	Name  string `yaml:"name,omitempty"`
}

type Host struct {
	SocketAddress SocketAddress `yaml:"socket_address,omitempty"`
}

type SocketAddress struct {
	Address   string `yaml:"address,omitempty"`
	PortValue string `yaml:"port_value,omitempty"`
}

type Listener struct {
	Address      Address       `yaml:"address,omitempty"`
	FilterChains []FilterChain `yaml:"filter_chains,omitempty"`
}

type Address struct {
	SocketAddress SocketAddress `yaml:"socket_address,omitempty"`
}

type FilterChain struct {
	Filters []Filter `yaml:"filters,omitempty"`
}

type Filter struct {
	Config Config `yaml:"config,omitempty"`
}

type Config struct {
	Cluster string `yaml:"cluster,omitempty"`
}

type BaseTemplate struct {
	UpstreamAddress, UpstreamPort, ListenerPort, Name, Key, Cert string
}

type Parser struct {
}

func NewParser() Parser {
	return Parser{}
}

/* Parses the Envoy conf file and extracts the clusters and a map of cluster names to listeners*/
// TODO: check if we can replace the multiple struct above
func getClusters(envoyConfFile string) (clusters []Cluster, nameToPortMap map[string]string, err error) {
	contents, err := ioutil.ReadFile(envoyConfFile)
	if err != nil {
		return []Cluster{}, map[string]string{}, fmt.Errorf("Failed to read envoy config: %s", err)
	}

	conf := EnvoyConf{}

	err = yaml.Unmarshal(contents, &conf)
	if err != nil {
		return []Cluster{}, map[string]string{}, fmt.Errorf("Failed to unmarshal envoy conf: %s", err)
	}

	for i := 0; i < len(conf.StaticResources.Clusters); i++ {
		clusters = append(clusters, conf.StaticResources.Clusters[i])
	}

	nameToPortMap = make(map[string]string)
	for i := 0; i < len(conf.StaticResources.Listeners); i++ {
		clusterName := conf.StaticResources.Listeners[i].FilterChains[0].Filters[0].Config.Cluster
		listenerPort := conf.StaticResources.Listeners[i].Address.SocketAddress.PortValue
		nameToPortMap[clusterName] = listenerPort
	}

	return clusters, nameToPortMap, nil
}

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
		return "", "", fmt.Errorf("Failed to read sds creds: %s", err)
	}

	auth := sds{}

	err = yaml.Unmarshal(contents, &auth)
	if err != nil {
		return "", "", fmt.Errorf("Failed to unmarshal sds creds: %s", err)
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

	clusters, nameToPortMap, err := getClusters(envoyConfFile)
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
		// TODO: add a test
		if err != nil {
			return err
		}
	}

	confTemplate := fmt.Sprintf(`
worker_processes  1;
daemon off;

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
		return err
	}

	err = ioutil.WriteFile(certFile, []byte(cert), 0644)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(keyFile, []byte(key), 0644)
	return err
}
