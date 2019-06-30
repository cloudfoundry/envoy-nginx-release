package app

import "strings"

const (
	DefaultEnvoyConfigPath          = "C:\\etc\\cf-assets\\envoy_config\\envoy.yaml"
	DefaultSdsCertAndKeysPath       = "C:\\etc\\cf-assets\\envoy_config\\sds-server-cert-and-keys.yaml"
	DefaultSdsValidationContextPath = "C:\\etc\\cf-assets\\envoy_config\\sds-server-validation-context.yml"
)

type Options struct {
	EnvoyConfig   string
	SdsCreds      string
	SdsValidation string
}

type Flags struct {
	options Options
}

func NewFlags() Flags {
	return Flags{
		options: Options{
			EnvoyConfig:   DefaultEnvoyConfigPath,
			SdsCreds:      DefaultSdsCertAndKeysPath,
			SdsValidation: DefaultSdsValidationContextPath,
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
		case "--creds":
			if hasValidArgument(i, args) {
				f.options.SdsCreds = args[i+1]
			}
		case "--validation":
			if hasValidArgument(i, args) {
				f.options.SdsValidation = args[i+1]
			}
		}
	}
	return f.options
}

func hasValidArgument(i int, args []string) bool {
	return (i+1) < len(args) && !strings.HasPrefix(args[i+1], "-")
}
