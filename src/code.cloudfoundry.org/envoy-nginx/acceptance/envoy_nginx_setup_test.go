package acceptance_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "code.cloudfoundry.org/envoy-nginx/testhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

const (
	SdsIdCredsFixture      = "../fixtures/cf_assets_envoy_config/sds-id-cert-and-key.yaml"
	SdsC2CCredsFixture     = "../fixtures/cf_assets_envoy_config/sds-c2c-cert-and-key.yaml"
	SdsIdValidationFixture = "../fixtures/cf_assets_envoy_config/sds-id-validation-context.yaml"
	EnvoyFixture           = "../fixtures/cf_assets_envoy_config/envoy.yaml"
)

var _ = Describe("Acceptance", func() {
	var (
		envoyNginxBin   string
		binParentDir    string
		sdsIdCredsFile  string
		sdsC2CCredsFile string
		cmd             *exec.Cmd
	)

	BeforeEach(func() {
		bin, err := gexec.Build("code.cloudfoundry.org/envoy-nginx")
		Expect(err).ToNot(HaveOccurred())

		binParentDir, err = ioutil.TempDir("", "envoy-nginx")
		Expect(err).ToNot(HaveOccurred())

		basename := filepath.Base(bin)
		err = os.Rename(bin, filepath.Join(binParentDir, basename))
		Expect(err).ToNot(HaveOccurred())
		envoyNginxBin = filepath.Join(binParentDir, basename)

		tmp, err := ioutil.TempFile("", "sdsIdCreds")
		Expect(err).ToNot(HaveOccurred())
		sdsIdCredsFile = tmp.Name()
		tmp.Close()
		err = CopyFile(SdsIdCredsFixture, sdsIdCredsFile)
		Expect(err).ToNot(HaveOccurred())

		tmp, err = ioutil.TempFile("", "sdsC2CCreds")
		Expect(err).ToNot(HaveOccurred())
		sdsC2CCredsFile = tmp.Name()
		tmp.Close()
		err = CopyFile(SdsC2CCredsFixture, sdsC2CCredsFile)
		Expect(err).ToNot(HaveOccurred())

		cmd = exec.Command(envoyNginxBin, "-c", EnvoyFixture, "--id-creds", sdsIdCredsFile, "--c2c-creds", sdsC2CCredsFile, "--id-validation", SdsIdValidationFixture)
	})

	AfterEach(func() {
		Expect(os.Remove(sdsIdCredsFile)).NotTo(HaveOccurred())
		Expect(os.Remove(sdsC2CCredsFile)).NotTo(HaveOccurred())
		Expect(os.RemoveAll(binParentDir)).NotTo(HaveOccurred())
	})

	Context("when nginx.exe is present in the same directory", func() {
		var (
			args     []string
			nginxDir string
			session  *gexec.Session
		)

		BeforeEach(func() {
			nginxBin, err := gexec.Build("code.cloudfoundry.org/envoy-nginx/fixtures/nginx")
			Expect(err).ToNot(HaveOccurred())

			err = os.Rename(nginxBin, filepath.Join(binParentDir, "nginx.exe"))
			Expect(err).ToNot(HaveOccurred())

			session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())

			// The output of the "fake" nginx.exe will always have a comma
			Eventually(session.Out).Should(gbytes.Say(","))
			args = strings.Split(string(session.Out.Contents()), ",")
			Expect(len(args)).To(Equal(3))

			nginxDir = findNginxConfDir(args)
			Expect(nginxDir).ToNot(BeEmpty())
		})

		AfterEach(func() {
			session.Terminate()
			Eventually(session, "5s").Should(gexec.Exit())

			Expect(os.RemoveAll(nginxDir)).NotTo(HaveOccurred())

			gexec.CleanupBuildArtifacts()
		})

		Context("when the sds file is rotated", func() {
			It("rewrites the id cert and key file and reloads nginx", func() {
				err := RotateCert("../fixtures/cf_assets_envoy_config/sds-id-cert-and-key-rotated.yaml", sdsIdCredsFile)
				Expect(err).ToNot(HaveOccurred())

				Eventually(session.Out).Should(gbytes.Say("detected change in sdsfile"))
				Eventually(session.Out).Should(gbytes.Say(fmt.Sprintf("-p,%s,-s,reload", strings.Replace(nginxDir, `\`, `\\`, -1))))

				expectedCert := `-----BEGIN CERTIFICATE-----
<<NEW EXPECTED ID CERT 1>>
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
<<NEW EXPECTED ID CERT 2>>
-----END CERTIFICATE-----
`
				expectedKey := `-----BEGIN RSA PRIVATE KEY-----
<<NEW EXPECTED ID KEY>>
-----END RSA PRIVATE KEY-----
`
				certFile := filepath.Join(nginxDir, "id-cert.pem")
				keyFile := filepath.Join(nginxDir, "id-key.pem")

				currentCert, err := ioutil.ReadFile(string(certFile))
				Expect(err).ShouldNot(HaveOccurred())

				currentKey, err := ioutil.ReadFile(string(keyFile))
				Expect(err).ShouldNot(HaveOccurred())

				Expect(string(currentCert)).To(Equal(expectedCert))
				Expect(string(currentKey)).To(Equal(expectedKey))
			})

			It("rewrites the id cert and key file and reloads nginx", func() {
				err := RotateCert("../fixtures/cf_assets_envoy_config/sds-c2c-cert-and-key-rotated.yaml", sdsC2CCredsFile)
				Expect(err).ToNot(HaveOccurred())

				Eventually(session.Out).Should(gbytes.Say("detected change in sdsfile"))
				Eventually(session.Out).Should(gbytes.Say(fmt.Sprintf("-p,%s,-s,reload", strings.Replace(nginxDir, `\`, `\\`, -1))))

				expectedCert := `-----BEGIN CERTIFICATE-----
<<NEW EXPECTED C2C CERT 1>>
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
<<NEW EXPECTED C2C CERT 2>>
-----END CERTIFICATE-----
`
				expectedKey := `-----BEGIN RSA PRIVATE KEY-----
<<NEW EXPECTED C2C KEY>>
-----END RSA PRIVATE KEY-----
`
				certFile := filepath.Join(nginxDir, "c2c-cert.pem")
				keyFile := filepath.Join(nginxDir, "c2c-key.pem")

				currentCert, err := ioutil.ReadFile(string(certFile))
				Expect(err).ShouldNot(HaveOccurred())

				currentKey, err := ioutil.ReadFile(string(keyFile))
				Expect(err).ShouldNot(HaveOccurred())

				Expect(string(currentCert)).To(Equal(expectedCert))
				Expect(string(currentKey)).To(Equal(expectedKey))
			})
		})
	})
})

func findNginxConfDir(args []string) string {
	nginxDir := ""
	for i, arg := range args {
		if arg == "-p" && len(args) > i+1 {
			nginxDir = strings.TrimSpace(args[i+1])
		}
	}
	return nginxDir
}
