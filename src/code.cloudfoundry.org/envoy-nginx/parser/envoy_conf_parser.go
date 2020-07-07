package parser

import (
	"fmt"
	"io/ioutil"
	"strings"

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
	Name           string         `yaml:"name,omitempty"`
	LoadAssignment LoadAssignment `yaml:"load_assignment,omitempty"`
}

type LoadAssignment struct {
	Endpoints []Endpoints `yaml:"endpoints,omitempty"`
}

type Endpoints struct {
	LBEndpoints []LBEndpoints `yaml:"lb_endpoints,omitempty"`
}

type LBEndpoints struct {
	Endpoint Endpoint `yaml:"endpoint,omitempty"`
}

type Endpoint struct {
	Address Address `yaml:"address,omitempty"`
}

type Listener struct {
	Address      Address       `yaml:"address,omitempty"`
	FilterChains []FilterChain `yaml:"filter_chains,omitempty"`
}

type Address struct {
	SocketAddress SocketAddress `yaml:"socket_address,omitempty"`
}

type SocketAddress struct {
	Address   string `yaml:"address,omitempty"`
	PortValue string `yaml:"port_value,omitempty"`
}

type FilterChain struct {
	Filters         []Filter        `yaml:"filters,omitempty"`
	TransportSocket TransportSocket `yaml:"transport_socket,omitempty"`
}

type Filter struct {
	TypedConfig TypedConfigTcpProxy `yaml:"typed_config,omitempty"`
}

type TypedConfigTcpProxy struct {
	Cluster string `yaml:"cluster,omitempty"`
}

type TransportSocket struct {
	TypedConfig TypedConfigDownstreamTlsContext `yaml:"typed_config,omitempty"`
}

type TypedConfigDownstreamTlsContext struct {
	CommonTLSContext         CommonTLSContext `yaml:"common_tls_context,omitempty"`
	RequireClientCertificate bool             `yaml:"require_client_certificate,omitempty"`
}

type CommonTLSContext struct {
	TLSParams TLSParams `yaml:"tls_params,omitempty"`
}

type TLSParams struct {
	CipherSuites []string `yaml:"cipher_suites,omitempty"`
}

type PortAndCiphers struct {
	Port    string
	Ciphers string
}

type EnvoyConfParser struct{}

func NewEnvoyConfParser() EnvoyConfParser {
	return EnvoyConfParser{}
}

// Read Envoy conf file and unmarshal it
func (e EnvoyConfParser) ReadUnmarshalEnvoyConfig(envoyConfFile string) (EnvoyConf, error) {
	conf := EnvoyConf{}

	contents, err := ioutil.ReadFile(envoyConfFile)
	if err != nil {
		return conf, fmt.Errorf("Failed to read envoy config: %s", err)
	}

	err = yaml.Unmarshal(contents, &conf)
	if err != nil {
		return conf, fmt.Errorf("Failed to unmarshal envoy config: %s", err)
	}

	return conf, nil
}

// Parses the Envoy conf file and extracts the clusters and a map of cluster names to listeners
func (e EnvoyConfParser) GetClusters(conf EnvoyConf) (clusters []Cluster, nameToPortAndCiphersMap map[string]PortAndCiphers) {
	for i := 0; i < len(conf.StaticResources.Clusters); i++ {
		clusters = append(clusters, conf.StaticResources.Clusters[i])
	}

	nameToPortAndCiphersMap = make(map[string]PortAndCiphers)
	for i := 0; i < len(conf.StaticResources.Listeners); i++ {
		clusterName := conf.StaticResources.Listeners[i].FilterChains[0].Filters[0].TypedConfig.Cluster
		listenerPort := conf.StaticResources.Listeners[i].Address.SocketAddress.PortValue

		ciphersArray := conf.StaticResources.Listeners[i].FilterChains[0].TransportSocket.TypedConfig.CommonTLSContext.TLSParams.CipherSuites
		ciphers := strings.Join(ciphersArray, ":")
		nameToPortAndCiphersMap[clusterName] = PortAndCiphers{listenerPort, ciphers}
	}

	return clusters, nameToPortAndCiphersMap
}

// Checks if MTLS is enabled in the Envoy conf file.
// Defaults to returning false if require_client_certificate isn't set.
func (e EnvoyConfParser) GetMTLS(conf EnvoyConf) bool {
	for _, listener := range conf.StaticResources.Listeners {
		for _, filterChain := range listener.FilterChains {
			// Return the first value of require_client_certificate.
			// If we ever expect these values to be different between listeners, we can deal with it then.
			return filterChain.TransportSocket.TypedConfig.RequireClientCertificate
		}
	}

	return false
}
