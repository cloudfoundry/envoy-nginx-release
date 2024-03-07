package parser

import (
	"errors"
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v2"
)

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

type sdsCredParser struct {
	file       string
	configType SdsConfigType
}

func NewSdsIdCredParser(file string) SdsCredParser {
	return sdsCredParser{
		file:       file,
		configType: SdsIdConfigType,
	}
}

func NewSdsC2CCredParser(file string) SdsCredParser {
	return sdsCredParser{
		file:       file,
		configType: SdsC2CConfigType,
	}
}

/* Parses the Envoy SDS file and extracts the cert and key */
func (p sdsCredParser) GetCertAndKey() (string, string, error) {
	contents, err := os.ReadFile(p.file)
	if err != nil {
		return "", "", fmt.Errorf("Failed to read sds creds: %s", err)
	}

	auth := Sds{}

	err = yaml.Unmarshal(contents, &auth)
	if err != nil {
		return "", "", fmt.Errorf("Failed to unmarshal sds creds: %s", err)
	}

	if len(auth.Resources) < 1 {
		return "", "", errors.New("resources section not found in sds cred file")
	}

	cert := auth.Resources[0].TLSCertificate.CertChain.InlineString
	key := auth.Resources[0].TLSCertificate.PrivateKey.InlineString

	return cert, key, nil
}

func (p sdsCredParser) ConfigType() SdsConfigType {
	return p.configType
}
