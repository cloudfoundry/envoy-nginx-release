package parser

import (
	"errors"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

/*
* TODO: Try to use this auth_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"?
 */
type SdsServerValidation struct {
	Resources []ValidationResource `yaml:"resources,omitempty"`
}

type ValidationResource struct {
	ValidationContext ValidationContext `yaml:"validation_context,omitempty"`
}

type ValidationContext struct {
	TrustedCA TrustedCA `yaml:"trusted_ca,omitempty"`
}

type TrustedCA struct {
	InlineString string `yaml:"inline_string,omitempty"`
}

type SdsServerValidationParser struct{}

func NewSdsServerValidationParser() SdsServerValidationParser {
	return SdsServerValidationParser{}
}

func (p SdsServerValidationParser) GetCACert(sdsFile string) (string, error) {
	contents, err := ioutil.ReadFile(sdsFile)
	if err != nil {
		return "", fmt.Errorf("Failed to read sds server validation context: open not-a-real-file: %s", err)
	}

	auth := SdsServerValidation{}

	err = yaml.Unmarshal(contents, &auth)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal sds server validation context: %s", err)
	}

	if len(auth.Resources) < 1 {
		return "", errors.New("resources section not found in sds-server-validation-context.yaml")
	}

	ca := auth.Resources[0].ValidationContext.TrustedCA.InlineString

	return ca, nil
}
