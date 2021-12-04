package app

import "strings"

const (
	DefaultEnvoyConfigPath            = "C:\\etc\\cf-assets\\envoy_config\\envoy.yaml"
	DefaultSdsIdCertAndKeysPath       = "C:\\etc\\cf-assets\\envoy_config\\sds-id-cert-and-key.yaml"
	DefaultSdsC2CCertAndKeysPath      = "C:\\etc\\cf-assets\\envoy_config\\sds-c2c-cert-and-key.yaml"
	DefaultSdsIdValidationContextPath = "C:\\etc\\cf-assets\\envoy_config\\sds-id-validation-context.yaml"
)

type Options struct {
	EnvoyConfig     string
	SdsIdCreds      string
	SdsC2CCreds     string
	SdsIdValidation string
}

type Flags struct {
	options Options
}

func NewFlags() Flags {
	return Flags{
		options: Options{
			EnvoyConfig:     DefaultEnvoyConfigPath,
			SdsIdCreds:      DefaultSdsIdCertAndKeysPath,
			SdsC2CCreds:     DefaultSdsC2CCertAndKeysPath,
			SdsIdValidation: DefaultSdsIdValidationContextPath,
		},
	}
}

func (f Flags) Parse(args []string) Options {
	for i, arg := range args {
		switch arg {
		case "-c":
			if hasValidArgument(i, args) {
				f.options.EnvoyConfig = args[i+1]
			}
		case "--id-creds":
			if hasValidArgument(i, args) {
				f.options.SdsIdCreds = args[i+1]
			}
		case "--c2c-creds":
			if hasValidArgument(i, args) {
				f.options.SdsC2CCreds = args[i+1]
			}
		case "--id-validation":
			if hasValidArgument(i, args) {
				f.options.SdsIdValidation = args[i+1]
			}
		}
	}
	return f.options
}

func hasValidArgument(i int, args []string) bool {
	return (i+1) < len(args) && !strings.HasPrefix(args[i+1], "-")
}
