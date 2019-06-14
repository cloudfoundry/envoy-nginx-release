package parser_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "code.cloudfoundry.org/envoy-nginx/parser"
)

var _ = Describe("Parser", func() {
	var envoyConfFile string
	var sdsCredsFile string
	var tmpdir string
	var err error
	var configFile string
	var config []byte

	Describe("GenerateConf", func() {
		BeforeEach(func() {
			envoyConfFile = "../fixtures/cf_assets_envoy_config/envoy.yaml"
			sdsCredsFile = "../fixtures/cf_assets_envoy_config/sds-server-cert-and-key.yaml"

			tmpdir, err = ioutil.TempDir("", "conf")
			Expect(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		Describe("Good configuration", func() {
			BeforeEach(func() {
				err = GenerateConf(envoyConfFile, sdsCredsFile, tmpdir)
				Expect(err).ShouldNot(HaveOccurred())

				configFile = filepath.Join(tmpdir, "envoy_nginx.conf")
				config, err = ioutil.ReadFile(configFile)
				Expect(err).ShouldNot(HaveOccurred())
			})

			Context("when envoyConf and sdsCreds files are configured correctly", func() {

				It("should generate a valid nginx.conf of non-zero size", func() {
					f, err := os.Stat(configFile)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(f.Size()).Should(BeNumerically(">", 0))
				})

				Describe("nginx.conf contents", func() {
					It("should have a valid pid directive", func() {
						// e.g. pid /Temp/nginx_024.pid;
						re := regexp.MustCompile(`[\r\n]pid\s*[\w/.]+;`)
						match := re.Find(config)
						Expect(match).NotTo(BeNil())
					})

					It("should have an upstream server with addr 172.30.2.245:8080", func() {
						re := regexp.MustCompile(`[\r\n]\s*server\s*172.30.2.245:8080;`)
						match := re.Find(config)
						Expect(match).NotTo(BeNil())
					})

					It("should have an upstream server with addr 172.30.2.245:2222", func() {
						re := regexp.MustCompile(`[\r\n]\s*server\s*172.30.2.245:2222;`)
						match := re.Find(config)
						Expect(match).NotTo(BeNil())
					})

					It("should have an upstream server with addr 172.30.2.245:1234", func() {
						re := regexp.MustCompile(`[\r\n]\s*server\s*172.30.2.245:1234;`)
						match := re.Find(config)
						Expect(match).NotTo(BeNil())
					})

					It("should have a valid server listening on 61001", func() {
						re := regexp.MustCompile(`[\r\n]\s*listen\s*61001\s*ssl;`)
						match := re.Find(config)
						Expect(match).NotTo(BeNil())

						re = regexp.MustCompile(`[\r\n]\s*proxy_pass\s*0-service-cluster;`)
						match = re.Find(config)
						Expect(match).NotTo(BeNil())
					})

					It("should have a valid server listening on 61002", func() {
						re := regexp.MustCompile(`[\r\n]\s*listen\s*61002\s*ssl;`)
						match := re.Find(config)
						Expect(match).NotTo(BeNil())

						re = regexp.MustCompile(`[\r\n]\s*proxy_pass\s*1-service-cluster;`)
						match = re.Find(config)
						Expect(match).NotTo(BeNil())
					})

					It("should have a valid server listening on 61003", func() {
						re := regexp.MustCompile(`[\r\n]\s*listen\s*61003\s*ssl;`)
						match := re.Find(config)
						Expect(match).NotTo(BeNil())

						re = regexp.MustCompile(`[\r\n]\s*proxy_pass\s*2-service-cluster;`)
						match = re.Find(config)
						Expect(match).NotTo(BeNil())
					})

					It("should specify the ssl certificate and key", func() {
						// TODO: test this separately for each server that is listening
						certPath := filepath.Join(tmpdir, "cert.pem")
						matcher := fmt.Sprintf(`[\r\n]\s*ssl_certificate\s*%s;`, convertToUnixPath(certPath))
						re := regexp.MustCompile(matcher)
						sslCertLine := re.Find(config)
						Expect(sslCertLine).NotTo(BeNil())

						sslCertPath := filepath.Join(tmpdir, "cert.pem")
						sslCert, err := ioutil.ReadFile(string(sslCertPath))
						Expect(err).ShouldNot(HaveOccurred())

						expectedCert := `-----BEGIN CERTIFICATE-----
<<EXPECTED CERT 1>>
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
<<EXPECTED CERT 2>>
-----END CERTIFICATE-----
`
						Expect(string(sslCert)).To(Equal(expectedCert))

						keyPath := filepath.Join(tmpdir, "key.pem")
						matcher = fmt.Sprintf(`[\r\n]\s*ssl_certificate_key\s*%s;`, convertToUnixPath(keyPath))
						re = regexp.MustCompile(matcher)
						sslCertKeyLine := re.Find(config)
						Expect(sslCertKeyLine).NotTo(BeNil())

						sslCertKeyPath := filepath.Join(tmpdir, "key.pem")
						sslCertKey, err := ioutil.ReadFile(string(sslCertKeyPath))
						Expect(err).ShouldNot(HaveOccurred())

						expectedCertKey := `-----BEGIN RSA PRIVATE KEY-----
<<EXPECTED KEY>>
-----END RSA PRIVATE KEY-----
`
						Expect(string(sslCertKey)).To(Equal(expectedCertKey))
					})
				})
			})
		})

		Describe("Bad configuration", func() {
			Context("when envoyConf doesn't exist", func() {
				It("should return a read error", func() {
					err = GenerateConf("", sdsCredsFile, tmpdir)
					Expect(err).To(MatchError("Failed to read envoy config: open : no such file or directory"))
				})
			})

			Context("when sdsCreds doesn't exist", func() {
				It("should return a read error", func() {
					err = GenerateConf(envoyConfFile, "", tmpdir)
					Expect(err).To(MatchError("Failed to read sds creds: open : no such file or directory"))
				})
			})

			Context("when a listener port is missing for a cluster name", func() {
				It("should return a custom error", func() {
					envoyConfFile = "../fixtures/cf_assets_envoy_config/envoy-cluster-without-listener.yaml"

					err = GenerateConf(envoyConfFile, sdsCredsFile, tmpdir)
					Expect(err).To(MatchError("port is missing for cluster name banana"))
				})
			})

			Context("when envoy conf contains invalid yaml", func() {
				var invalidYamlFile string
				BeforeEach(func() {
					tmpFile, err := ioutil.TempFile(os.TempDir(), "envoy-invalid.yaml")
					Expect(err).NotTo(HaveOccurred())

					invalidYamlFile = tmpFile.Name()

					_, err = tmpFile.Write([]byte("%%%"))
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func() {
					os.Remove(invalidYamlFile)
				})

				Context("when envoy conf contents fail to unmarshal", func() {
					It("should return unmarshal error", func() {
						err = GenerateConf(invalidYamlFile, sdsCredsFile, tmpdir)
						Expect(err).To(MatchError("Failed to unmarshal envoy conf: yaml: could not find expected directive name"))
					})
				})

				Context("when sds creds contents fail to unmarshal", func() {
					It("should return unmarshal error", func() {
						err = GenerateConf(envoyConfFile, invalidYamlFile, tmpdir)
						Expect(err).To(MatchError("Failed to unmarshal sds creds: yaml: could not find expected directive name"))
					})
				})
			})
		})
	})
})

func convertToUnixPath(path string) string {
	path = strings.Replace(path, "C:", "", -1)
	path = strings.Replace(path, "\\", "/", -1)
	return path
}
