package acceptance_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Flags", func() {
	var (
		envoyNginxBin string
	)

	BeforeEach(func() {
		var err error
		envoyNginxBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		gexec.CleanupBuildArtifacts()
	})

	Context("when passed a flag it does not recognize", func() {
		It("does not fail with undefined flag error", func() {
			session, err := gexec.Start(exec.Command(envoyNginxBin, "-z", "nope"), GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).ShouldNot(gbytes.Say("Flags parse:"))
		})
	})
})
