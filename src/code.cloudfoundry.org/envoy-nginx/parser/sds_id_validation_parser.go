package parser

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type SdsIdValidation struct {
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

type sdsIdValidationParser struct {
	file string
}

func NewSdsIdValidationParser(file string) SdsValidationParser {
	return sdsIdValidationParser{
		file: file,
	}
}

func (p sdsIdValidationParser) GetCACert() (string, error) {
	contents, err := os.ReadFile(p.file)
	if err != nil {
		return "", fmt.Errorf("Failed to read sds server validation context: %s", err)
	}

	auth := SdsIdValidation{}

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
