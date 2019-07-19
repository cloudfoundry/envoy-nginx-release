package parser_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var EnvoyConfigFixture = "../fixtures/cf_assets_envoy_config/envoy.yaml"

func TestParser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Parser Suite")
}
