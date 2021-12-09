package fakes

import "code.cloudfoundry.org/envoy-nginx/parser"

type SdsCredParser struct {
	GetCertAndKeyCall struct {
		CallCount int
		Returns   struct {
			Cert  string
			Key   string
			Error error
		}
	}

	ConfigTypeCall struct {
		CallCount int
		Returns   struct {
			ConfigType parser.SdsConfigType
		}
	}
}

func (e SdsCredParser) GetCertAndKey() (string, string, error) {
	e.GetCertAndKeyCall.CallCount++

	return e.GetCertAndKeyCall.Returns.Cert, e.GetCertAndKeyCall.Returns.Key, e.GetCertAndKeyCall.Returns.Error
}

func (e SdsCredParser) ConfigType() parser.SdsConfigType {
	e.ConfigTypeCall.CallCount++

	return e.ConfigTypeCall.Returns.ConfigType
}
