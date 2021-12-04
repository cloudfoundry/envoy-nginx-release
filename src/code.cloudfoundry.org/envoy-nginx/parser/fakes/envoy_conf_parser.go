package fakes

import (
	"code.cloudfoundry.org/envoy-nginx/parser"
)

type EnvoyConfParser struct {
	ReadUnmarshalEnvoyConfigCall struct {
		CallCount int
		Receives  struct {
			EnvoyConfFile string
		}
		Returns struct {
			EnvoyConf parser.EnvoyConf
			Error     error
		}
	}

	GetClustersCall struct {
		CallCount int
		Receives  struct {
			EnvoyConf parser.EnvoyConf
		}
		Returns struct {
			Clusters        []parser.Cluster
			NameToListeners map[string][]parser.ListenerInfo
		}
	}
}

func (e *EnvoyConfParser) ReadUnmarshalEnvoyConfig(envoyConfFile string) (parser.EnvoyConf, error) {
	e.ReadUnmarshalEnvoyConfigCall.CallCount++
	e.ReadUnmarshalEnvoyConfigCall.Receives.EnvoyConfFile = envoyConfFile

	return e.ReadUnmarshalEnvoyConfigCall.Returns.EnvoyConf, e.ReadUnmarshalEnvoyConfigCall.Returns.Error
}

func (e *EnvoyConfParser) GetClusters(envoyConf parser.EnvoyConf) ([]parser.Cluster, map[string][]parser.ListenerInfo) {
	e.GetClustersCall.CallCount++
	e.GetClustersCall.Receives.EnvoyConf = envoyConf

	return e.GetClustersCall.Returns.Clusters, e.GetClustersCall.Returns.NameToListeners
}
