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
	SdsCredsFixture      = "../fixtures/cf_assets_envoy_config/sds-server-cert-and-key.yaml"
	SdsValidationFixture = "../fixtures/cf_assets_envoy_config/sds-server-validation-context.yaml"
	EnvoyFixture         = "../fixtures/cf_assets_envoy_config/envoy.yaml"
)

var _ = Describe("Acceptance", func() {
	var (
		envoyNginxBin     string
		binParentDir      string
		sdsCredsFile      string
		sdsValidationFile string
		cmd               *exec.Cmd
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

		tmp, err := ioutil.TempFile("", "sdsCreds")
		Expect(err).ToNot(HaveOccurred())
		sdsCredsFile = tmp.Name()
		tmp.Close()
		err = CopyFile(SdsCredsFixture, sdsCredsFile)
		Expect(err).ToNot(HaveOccurred())

		tmp, err = ioutil.TempFile("", "sdsValidation")
		Expect(err).ToNot(HaveOccurred())
		sdsValidationFile = tmp.Name()
		tmp.Close()
		err = CopyFile(SdsValidationFixture, sdsValidationFile)
		Expect(err).ToNot(HaveOccurred())

		cmd = exec.Command(envoyNginxBin, "-c", EnvoyFixture, "--creds", sdsCredsFile, "--validation", sdsValidationFile)
	})

	AfterEach(func() {
		Expect(os.Remove(sdsCredsFile)).NotTo(HaveOccurred())

		Expect(os.Remove(sdsValidationFile)).NotTo(HaveOccurred())

		Expect(os.RemoveAll(binParentDir)).NotTo(HaveOccurred())
	})

	Context("when nginx.exe is present in the same directory", func() {
		var (
			args    []string
			confDir string
			session *gexec.Session
		)

		BeforeEach(func() {
			nginxBin, err := gexec.Build("code.cloudfoundry.org/envoy-nginx/fixtures/nginx")
			Expect(err).ToNot(HaveOccurred())

			err = os.Rename(nginxBin, filepath.Join(binParentDir, "nginx.exe"))
			Expect(err).ToNot(HaveOccurred())
			nginxBin = filepath.Join(binParentDir, "nginx.exe")

			session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())

			// The output of the "fake" nginx.exe will always have a comma
			Eventually(session.Out).Should(gbytes.Say(","))
			args = strings.Split(string(session.Out.Contents()), ",")
			Expect(len(args)).To(Equal(5))

			confDir = findNginxConfDir(args)
			Expect(confDir).ToNot(BeEmpty())
		})

		AfterEach(func() {
			session.Terminate()
			Eventually(session, "5s").Should(gexec.Exit())

			Expect(os.RemoveAll(confDir)).NotTo(HaveOccurred())

			gexec.CleanupBuildArtifacts()
		})

		Context("when the sds file is rotated", func() {
			It("rewrites the cert and key file and reloads nginx", func() {
				err := RotateCert("../fixtures/cf_assets_envoy_config/sds-server-cert-and-key-rotated.yaml", sdsCredsFile)
				Expect(err).ToNot(HaveOccurred())

				nginxConf := strings.Replace(filepath.Join(confDir, "envoy_nginx.conf"), `\`, `\\`, -1)
				Eventually(session.Out).Should(gbytes.Say(fmt.Sprintf("-c,%s,-p,%s,-s,reload", nginxConf, strings.Replace(confDir, `\`, `\\`, -1))))

				expectedCert := `-----BEGIN CERTIFICATE-----
<<NEW EXPECTED CERT 1>>
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
<<NEW EXPECTED CERT 2>>
-----END CERTIFICATE-----
`
				expectedKey := `-----BEGIN RSA PRIVATE KEY-----
<<NEW EXPECTED KEY>>
-----END RSA PRIVATE KEY-----
`
				certFile := filepath.Join(confDir, "cert.pem")
				keyFile := filepath.Join(confDir, "key.pem")

				currentCert, err := ioutil.ReadFile(string(certFile))
				Expect(err).ShouldNot(HaveOccurred())

				currentKey, err := ioutil.ReadFile(string(keyFile))
				Expect(err).ShouldNot(HaveOccurred())

				Expect(string(currentCert)).To(Equal(expectedCert))
				Expect(string(currentKey)).To(Equal(expectedKey))
			})
		})

		It("calls nginx with the right arguments", func() {
			Expect(strings.TrimSpace(args[1])).To(Equal("-c"))

			nginxConf := strings.TrimSpace(args[2])
			_, err := os.Stat(nginxConf)
			Expect(err).ToNot(HaveOccurred())

			Expect(strings.TrimSpace(args[3])).To(Equal("-p"))

			Expect(confDir).ToNot(BeEmpty())
		})

		It("creates the right files in the output directory", func() {
			files, err := ioutil.ReadDir(confDir)
			Expect(err).ToNot(HaveOccurred())

			names := []string{}
			for _, file := range files {
				names = append(names, file.Name())
			}
			Expect(names).To(ConsistOf("logs", "envoy_nginx.conf", "cert.pem", "key.pem", "ca.pem"))
		})
	})

	Context("nginx.exe fails when reloaded", func() {
		BeforeEach(func() {
			nginxBin, err := gexec.Build("code.cloudfoundry.org/envoy-nginx/fixtures/bad-nginx-reload")
			Expect(err).ToNot(HaveOccurred())

			err = os.Rename(nginxBin, filepath.Join(binParentDir, "nginx.exe"))
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			// TODO: this test is orphaning the nginx-conf temporary directoy. Need to clean it up.
		})

		It("exits with error", func() {
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())

			// The output of the "fake" nginx.exe will always have a comma
			// Include this line so that the file watcher will have a chance to start
			Eventually(session.Out).Should(gbytes.Say(","))

			By("simulating the cert/key rotation by diego")
			err = RotateCert("../fixtures/cf_assets_envoy_config/sds-server-cert-and-key-rotated.yaml", sdsCredsFile)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session.Out).Should(gbytes.Say("-s,reload"))

			// TODO: Validate what this error is and what the error message looks like for users.
			Eventually(session, "5s").Should(gexec.Exit(1))
		})
	})

	Context("when nginx.exe fails", func() {
		BeforeEach(func() {
			nginxBin, err := gexec.Build("code.cloudfoundry.org/envoy-nginx/fixtures/bad-nginx")
			Expect(err).ToNot(HaveOccurred())

			err = os.Rename(nginxBin, filepath.Join(binParentDir, "nginx.exe"))
			Expect(err).ToNot(HaveOccurred())

			nginxBin = filepath.Join(binParentDir, "nginx.exe")
		})

		AfterEach(func() {
			// TODO: this test is orphaning the nginx-conf temporary directoy. Need to clean it up.
		})

		It("returns the error", func() {
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "2s").Should(gexec.Exit(1))
			Eventually(session).Should(gbytes.Say("envoy.exe: Executing: "))
		})
	})

	Context("when nginx.exe is not present in the same directory", func() {
		var (
			aloneBin       string
			aloneParentDir string
		)

		BeforeEach(func() {
			var err error
			aloneBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx")
			Expect(err).ToNot(HaveOccurred())

			aloneParentDir, err = ioutil.TempDir("", "envoy-nginx")
			Expect(err).ToNot(HaveOccurred())

			basename := filepath.Base(aloneBin)
			err = os.Rename(aloneBin, filepath.Join(aloneParentDir, basename))
			Expect(err).ToNot(HaveOccurred())

			aloneBin = filepath.Join(aloneParentDir, basename)
		})

		It("returns a helpful error message", func() {
			session, err := gexec.Start(exec.Command(aloneBin, "-c", EnvoyFixture), GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gbytes.Say("envoy-nginx application: get nginx-path: stat nginx.exe:"))
			Eventually(session, "2s").ShouldNot(gexec.Exit(0))
		})

		AfterEach(func() {
			os.RemoveAll(aloneParentDir)
		})
	})
})

func findNginxConfDir(args []string) string {
	confDir := ""
	for i, arg := range args {
		if arg == "-p" && len(args) > i+1 {
			confDir = strings.TrimSpace(args[i+1])
		}
	}
	return confDir
}
