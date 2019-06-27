package acceptance_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gexec"
)

const EnvoyFile = "../fixtures/cf_assets_envoy_config/envoy.yaml"

var _ = Describe("Flags", func() {
	var envoyNginxBin string

	BeforeEach(func() {
		var err error
		envoyNginxBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx")
		Expect(err).ToNot(HaveOccurred())
	})

	PIt("accepts overrides for the flags", func() {
		_, err := gexec.Start(exec.Command(envoyNginxBin, "-c", EnvoyFile), GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
	})
})
