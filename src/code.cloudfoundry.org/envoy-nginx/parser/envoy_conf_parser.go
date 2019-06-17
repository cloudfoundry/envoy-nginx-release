package parser

import (
	"fmt"
	"io/ioutil"

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

type EnvoyConfParser struct{}

func NewEnvoyConfParser() EnvoyConfParser {
	return EnvoyConfParser{}
}

/* Parses the Envoy conf file and extracts the clusters and a map of cluster names to listeners*/
func (e EnvoyConfParser) GetClusters(envoyConfFile string) (clusters []Cluster, nameToPortMap map[string]string, err error) {
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
