package parser

import (
	"errors"
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

/*
* TODO: Try to use this auth_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"?
 */
type Sds struct {
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

type SdsCredParser struct{}

func NewSdsCredParser() SdsCredParser {
	return SdsCredParser{}
}

/* Parses the Envoy SDS file and extracts the cert and key */
func (p SdsCredParser) GetCertAndKey(sdsFile string) (string, string, error) {
	contents, err := ioutil.ReadFile(sdsFile)
	if err != nil {
		return "", "", fmt.Errorf("Failed to read sds creds: %s", err)
	}

	auth := Sds{}

	err = yaml.Unmarshal(contents, &auth)
	if err != nil {
		return "", "", fmt.Errorf("Failed to unmarshal sds creds: %s", err)
	}

	if len(auth.Resources) < 1 {
		return "", "", errors.New("resources section not found in sds-server-cert-and-key.yaml")
	}

	cert := auth.Resources[0].TLSCertificate.CertChain.InlineString
	key := auth.Resources[0].TLSCertificate.PrivateKey.InlineString

	return cert, key, nil
}
