package acceptance_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Flags", func() {
	var (
		envoyNginxBin string
		cmd           *exec.Cmd
	)

	BeforeEach(func() {
		var err error
		envoyNginxBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx")
		Expect(err).ToNot(HaveOccurred())

		cmd = exec.Command(envoyNginxBin, "-c", EnvoyFixture, "--id-creds", SdsIdCredsFixture, "--c2c-creds", SdsC2CCredsFixture, "--id-validation", SdsIdValidationFixture)
	})

	AfterEach(func() {
		gexec.CleanupBuildArtifacts()
	})

	PIt("accepts overrides for the flags", func() {
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gbytes.Say("hi"))
		Eventually(session).Should(gexec.Exit(0))
	})

	Context("when passed a flag it does not recognize", func() {
		It("does not fail with undefined flag error", func() {
			session, err := gexec.Start(exec.Command(envoyNginxBin, "-z", "nope"), GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).ShouldNot(gbytes.Say("Flags parse:"))
		})
	})
})
