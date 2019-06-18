package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const sdsFixture = "fixtures/cf_assets_envoy_config/sds-server-cert-and-key.yaml"

func TestEnvoyNginx(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EnvoyNginx Suite")
}
