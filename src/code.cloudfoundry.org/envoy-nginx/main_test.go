package main_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Envoy-Nginx", func() {
	Context("when nginx.exe is present in the same directory", func() {
		var (
			envoyNginxBin   string
			err             error
			binParentDir    string
			nginxBin        string
			args            []string
			outputDirectory string
			session         *gexec.Session
			sdsFile         string
		)

		BeforeEach(func() {
			envoyNginxBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx")
			Expect(err).ToNot(HaveOccurred())

			binParentDir, err = ioutil.TempDir("", "envoy-nginx")
			Expect(err).ToNot(HaveOccurred())

			basename := filepath.Base(envoyNginxBin)
			err = os.Rename(envoyNginxBin, filepath.Join(binParentDir, basename))
			Expect(err).ToNot(HaveOccurred())
			envoyNginxBin = filepath.Join(binParentDir, basename)
			os.Setenv("ENVOY_FILE", "fixtures/cf_assets_envoy_config/envoy.yaml")
			os.Setenv("SDS_FILE", "fixtures/cf_assets_envoy_config/sds-server-cert-and-key.yaml")

			nginxBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx/fixtures/nginx")
			Expect(err).ToNot(HaveOccurred())

			err = os.Rename(nginxBin, filepath.Join(binParentDir, "nginx.exe"))
			Expect(err).ToNot(HaveOccurred())
			nginxBin = filepath.Join(binParentDir, "nginx.exe")

			sdsFd, err := ioutil.TempFile("", "sdsFile")
			Expect(err).ToNot(HaveOccurred())
			sdsFile = sdsFd.Name()
			err = copyFile(sdsFixture, sdsFile)
			Expect(err).ToNot(HaveOccurred())
			os.Setenv("SDS_FILE", sdsFile)

			session, err = Start(exec.Command(envoyNginxBin))
			Expect(err).ToNot(HaveOccurred())

			// The output of the "fake" nginx.exe will always have a comma
			Eventually(session.Out).Should(gbytes.Say(","))
			args = strings.Split(string(session.Out.Contents()), ",")

			Expect(len(args)).To(Equal(5))

			/*
			* TODO: see about cleaning up output directory.
			* There's a risk that if we get it wrong, we end up deleting
			* some random directory on our filesystem.
			 */
			outputDirectory = strings.TrimSpace(args[4])
		})

		Context("when the sds file is rotated", func() {
			It("rewrites the cert and key file and reloads nginx", func() {
				rotateCert("fixtures/cf_assets_envoy_config/sds-server-cert-and-key-rotated.yaml", sdsFile)
				Eventually(session.Out).Should(gbytes.Say("-s,reload"))

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
				certFile := filepath.Join(outputDirectory, "cert.pem")
				keyFile := filepath.Join(outputDirectory, "key.pem")

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
			_, err = os.Stat(nginxConf)
			Expect(err).ToNot(HaveOccurred())

			Expect(strings.TrimSpace(args[3])).To(Equal("-p"))

			Expect(outputDirectory).ToNot(BeEmpty())
		})

		It("creates the right files in the output directory", func() {
			files, err := ioutil.ReadDir(outputDirectory)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(files)).To(Equal(3))

			var foundConf, foundCert, foundKey bool

			for _, file := range files {
				if file.Name() == "envoy_nginx.conf" {
					foundConf = true
				}
				if file.Name() == "cert.pem" {
					foundCert = true
				}
				if file.Name() == "key.pem" {
					foundKey = true
				}
			}

			Expect(foundConf).To(BeTrue())
			Expect(foundCert).To(BeTrue())
			Expect(foundKey).To(BeTrue())

			expectedCert := `-----BEGIN CERTIFICATE-----
<<EXPECTED CERT 1>>
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
<<EXPECTED CERT 2>>
-----END CERTIFICATE-----
`
			expectedKey := `-----BEGIN RSA PRIVATE KEY-----
<<EXPECTED KEY>>
-----END RSA PRIVATE KEY-----
`
			certFile := filepath.Join(outputDirectory, "cert.pem")
			keyFile := filepath.Join(outputDirectory, "key.pem")

			currentCert, err := ioutil.ReadFile(string(certFile))
			Expect(err).ShouldNot(HaveOccurred())

			currentKey, err := ioutil.ReadFile(string(keyFile))
			Expect(err).ShouldNot(HaveOccurred())

			Expect(string(currentCert)).To(Equal(expectedCert))
			Expect(string(currentKey)).To(Equal(expectedKey))
		})

		AfterEach(func() {
			os.RemoveAll(binParentDir)
			os.RemoveAll(outputDirectory)
		})
	})

	Context("nginx.exe fails when reloaded", func() {
		var (
			envoyNginxBin string
			err           error
			binParentDir  string
			nginxBin      string
			sdsFile       string
		)

		BeforeEach(func() {
			envoyNginxBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx")
			Expect(err).ToNot(HaveOccurred())

			binParentDir, err = ioutil.TempDir("", "envoy-nginx")
			Expect(err).ToNot(HaveOccurred())

			basename := filepath.Base(envoyNginxBin)
			err = os.Rename(envoyNginxBin, filepath.Join(binParentDir, basename))
			Expect(err).ToNot(HaveOccurred())
			envoyNginxBin = filepath.Join(binParentDir, basename)

			nginxBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx/fixtures/bad-nginx-reload")
			Expect(err).ToNot(HaveOccurred())

			err = os.Rename(nginxBin, filepath.Join(binParentDir, "nginx.exe"))
			Expect(err).ToNot(HaveOccurred())
			nginxBin = filepath.Join(binParentDir, "nginx.exe")

			sdsFd, err := ioutil.TempFile("", "sdsFile")
			Expect(err).ToNot(HaveOccurred())
			sdsFile = sdsFd.Name()
			err = copyFile(sdsFixture, sdsFile)
			Expect(err).ToNot(HaveOccurred())
			os.Setenv("SDS_FILE", sdsFile)

			os.Setenv("ENVOY_FILE", "fixtures/cf_assets_envoy_config/envoy.yaml")
		})

		Context("when nginx.exe fails when reloaded", func() {
			It("exits with error", func() {
				session, err := Start(exec.Command(envoyNginxBin))
				Expect(err).ToNot(HaveOccurred())

				// The output of the "fake" nginx.exe will always have a comma
				// Include this line so that the file watcher will have a chance to start
				Eventually(session.Out).Should(gbytes.Say(","))

				By("simulating the cert/key rotation by diego")
				rotateCert("fixtures/cf_assets_envoy_config/sds-server-cert-and-key-rotated.yaml", sdsFile)
				Eventually(session.Out).Should(gbytes.Say("-s,reload"))

				Eventually(session, "5s").Should(gexec.Exit())
				Expect(session.ExitCode()).ToNot(Equal(0))
			})
		})
	})

	Context("bad nginx.exe", func() {
		var (
			envoyNginxBin string
			err           error
			binParentDir  string
			nginxBin      string
			sdsFile       string
		)

		BeforeEach(func() {
			envoyNginxBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx")
			Expect(err).ToNot(HaveOccurred())

			binParentDir, err = ioutil.TempDir("", "envoy-nginx")
			Expect(err).ToNot(HaveOccurred())

			basename := filepath.Base(envoyNginxBin)
			err = os.Rename(envoyNginxBin, filepath.Join(binParentDir, basename))
			Expect(err).ToNot(HaveOccurred())
			envoyNginxBin = filepath.Join(binParentDir, basename)

			nginxBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx/fixtures/bad-nginx")
			Expect(err).ToNot(HaveOccurred())

			err = os.Rename(nginxBin, filepath.Join(binParentDir, "nginx.exe"))
			Expect(err).ToNot(HaveOccurred())
			nginxBin = filepath.Join(binParentDir, "nginx.exe")

			sdsFd, err := ioutil.TempFile("", "sdsFile")
			Expect(err).ToNot(HaveOccurred())
			sdsFile = sdsFd.Name()
			err = copyFile(sdsFixture, sdsFile)
			Expect(err).ToNot(HaveOccurred())
			os.Setenv("SDS_FILE", sdsFile)
		})

		Context("when nginx.exe exits with error", func() {
			It("exits with error", func() {
				_, _, err := Execute(exec.Command(envoyNginxBin))
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("when nginx.exe is not present in the same directory", func() {
		var (
			aloneBin       string
			aloneParentDir string
			err            error
		)

		BeforeEach(func() {
			aloneBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx")
			Expect(err).ToNot(HaveOccurred())

			aloneParentDir, err = ioutil.TempDir("", "envoy-nginx")
			Expect(err).ToNot(HaveOccurred())

			basename := filepath.Base(aloneBin)
			os.Rename(aloneBin, filepath.Join(aloneParentDir, basename))
			Expect(err).ToNot(HaveOccurred())

			aloneBin = filepath.Join(aloneParentDir, basename)
		})

		It("errors", func() {
			_, _, err = Execute(exec.Command(aloneBin))
			Expect(err).To(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(aloneParentDir)
		})
	})
})
