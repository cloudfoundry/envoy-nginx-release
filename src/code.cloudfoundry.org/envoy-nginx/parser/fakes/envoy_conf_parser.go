package fakes

import (
	"code.cloudfoundry.org/envoy-nginx/parser"
)

type EnvoyConfParser struct {
	GetClustersCall struct {
		CallCount int
		Receives  struct {
			EnvoyConfFile string
		}
		Returns struct {
			Clusters      []parser.Cluster
			NameToPortMap map[string]string
			Error         error
		}
	}

	GetMTLSCall struct {
		CallCount int
		Receives  struct {
			EnvoyConfFile string
		}
		Returns struct {
			MTLS  bool
			Error error
		}
	}
}

func (e *EnvoyConfParser) GetClusters(envoyConfFile string) ([]parser.Cluster, map[string]string, error) {
	e.GetClustersCall.CallCount++
	e.GetClustersCall.Receives.EnvoyConfFile = envoyConfFile

	return e.GetClustersCall.Returns.Clusters, e.GetClustersCall.Returns.NameToPortMap, e.GetClustersCall.Returns.Error
}

func (e *EnvoyConfParser) GetMTLS(envoyConfFile string) (bool, error) {
	e.GetMTLSCall.CallCount++
	e.GetMTLSCall.Receives.EnvoyConfFile = envoyConfFile

	return e.GetMTLSCall.Returns.MTLS, e.GetMTLSCall.Returns.Error
}
