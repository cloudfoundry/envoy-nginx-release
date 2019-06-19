/* Faker envoy.exe */
package main

import (
	"os"
)

const DefaultEnvoyConfFile = "C:\\etc\\cf-assets\\envoy_config\\envoy.yaml"

func main() {

	//TODO Find some stable flags library that lets you ignore unknown args
	//We can't handle all envoy args!
	envoyConfig := DefaultEnvoyConfFile
	for i, arg := range os.Args {
		if arg == "-c" && len(os.Args) > i+1 && os.Args[i+1] != "" {
			envoyConfig = os.Args[i+1]
		}
	}
	envoy(envoyConfig)
}
