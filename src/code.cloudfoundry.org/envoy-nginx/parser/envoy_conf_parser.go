package parser

import (
	"fmt"
	"io/ioutil"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type SdsConfigType int

const (
	SdsIdConfigType SdsConfigType = iota
	SdsC2CConfigType
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
	TLSCertificateSdsSecretConfigs []TLSCertificateSdsSecretConfig `yaml:"tls_certificate_sds_secret_configs,omitempty"`
	TLSParams                      TLSParams                       `yaml:"tls_params,omitempty"`
}

type TLSCertificateSdsSecretConfig struct {
	Name string `yaml:"name"`
}

type TLSParams struct {
	CipherSuites []string `yaml:"cipher_suites,omitempty"`
}

type ListenerInfo struct {
	Port          string
	MTLS          bool
	Ciphers       string
	SdsConfigType SdsConfigType
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
func (e EnvoyConfParser) GetClusters(conf EnvoyConf) (clusters []Cluster, nameToListeners map[string][]ListenerInfo) {
	for i := 0; i < len(conf.StaticResources.Clusters); i++ {
		clusters = append(clusters, conf.StaticResources.Clusters[i])
	}

	nameToListeners = make(map[string][]ListenerInfo)
	for i := 0; i < len(conf.StaticResources.Listeners); i++ {
		clusterName := conf.StaticResources.Listeners[i].FilterChains[0].Filters[0].TypedConfig.Cluster
		listenerPort := conf.StaticResources.Listeners[i].Address.SocketAddress.PortValue
		mTLS := conf.StaticResources.Listeners[i].FilterChains[0].TransportSocket.TypedConfig.RequireClientCertificate

		ciphersArray := conf.StaticResources.Listeners[i].FilterChains[0].TransportSocket.TypedConfig.CommonTLSContext.TLSParams.CipherSuites
		ciphers := strings.Join(ciphersArray, ":")

		var sdsConfigType SdsConfigType
		if conf.StaticResources.Listeners[i].FilterChains[0].TransportSocket.TypedConfig.CommonTLSContext.TLSCertificateSdsSecretConfigs[0].Name == "c2c-cert-and-key" {
			sdsConfigType = SdsC2CConfigType
		} else {
			sdsConfigType = SdsIdConfigType
		}
		nameToListeners[clusterName] = append(nameToListeners[clusterName], ListenerInfo{
			Port:          listenerPort,
			MTLS:          mTLS,
			Ciphers:       ciphers,
			SdsConfigType: sdsConfigType,
		})
	}

	return clusters, nameToListeners
}
