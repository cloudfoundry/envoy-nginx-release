package parser_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var EnvoyConfigFixture = "../fixtures/cf_assets_envoy_config/envoy.yaml"
var EnvoyOneListenerPerServerConfigFixture = "../fixtures/cf_assets_envoy_config/envoy_one_listener_per_server.yaml"

func TestParser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Parser Suite")
}
