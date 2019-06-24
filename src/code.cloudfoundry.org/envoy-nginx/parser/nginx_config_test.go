package parser_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/envoy-nginx/parser"
	"code.cloudfoundry.org/envoy-nginx/parser/fakes"
)

var _ = Describe("Nginx Config", func() {
	var (
		envoyConfFile     string
		sdsCredsFile      string
		sdsValidationFile string
		tmpdir            string
		configFile        string
		config            []byte

		envoyConfParser     *fakes.EnvoyConfParser
		nginxConfig         parser.NginxConfig
		sdsCredParser       *fakes.SdsCredParser
		sdsValidationParser *fakes.SdsServerValidationParser
	)

	BeforeEach(func() {
		envoyConfFile = "../fixtures/cf_assets_envoy_config/envoy.yaml"
		sdsCredsFile = "../fixtures/cf_assets_envoy_config/sds-server-cert-and-key.yaml"
		sdsValidationFile = "../fixtures/cf_assets_envoy_config/sds-server-validation-context.yaml"

		sdsCredParser = &fakes.SdsCredParser{}
		sdsValidationParser = &fakes.SdsServerValidationParser{}
		envoyConfParser = &fakes.EnvoyConfParser{}

		var err error
		tmpdir, err = ioutil.TempDir("", "conf")
		Expect(err).ShouldNot(HaveOccurred())

		nginxConfig = parser.NewNginxConfig(envoyConfParser, sdsCredParser, sdsValidationParser, tmpdir)
	})

	Describe("WriteTLSFiles", func() {
		BeforeEach(func() {
			sdsCredParser.GetCertAndKeyCall.Returns.Cert = "some-cert"
			sdsCredParser.GetCertAndKeyCall.Returns.Key = "some-key"
			sdsValidationParser.GetCACertCall.Returns.CA = "some-ca-cert"
		})

		It("should have written cert, key, and ca", func() {
			err := nginxConfig.WriteTLSFiles(sdsCredsFile, sdsValidationFile)
			Expect(err).ShouldNot(HaveOccurred())

			certPath := filepath.Join(tmpdir, "cert.pem")
			keyPath := filepath.Join(tmpdir, "key.pem")
			caPath := filepath.Join(tmpdir, "ca.pem")

			cert, err := ioutil.ReadFile(string(certPath))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(cert)).To(Equal("some-cert"))

			key, err := ioutil.ReadFile(string(keyPath))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(key)).To(Equal("some-key"))

			ca, err := ioutil.ReadFile(string(caPath))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(ca)).To(Equal("some-ca-cert"))
		})

		Context("when sds cred parser fails to get cert and key", func() {
			BeforeEach(func() {
				sdsCredParser.GetCertAndKeyCall.Returns.Error = errors.New("banana")
			})

			It("returns a helpful error message", func() {
				err := nginxConfig.WriteTLSFiles(sdsCredsFile, sdsValidationFile)
				Expect(err).To(MatchError("Failed to get cert and key from sds file: banana"))
			})
		})

		Context("when sds validation context parser fails to get ca", func() {
			BeforeEach(func() {
				sdsValidationParser.GetCACertCall.Returns.Error = errors.New("banana")
			})

			It("returns a helpful error message", func() {
				err := nginxConfig.WriteTLSFiles(sdsCredsFile, sdsValidationFile)
				Expect(err).To(MatchError("Failed to get ca cert from sds server validation context file: banana"))
			})
		})
	})

	Describe("Generate", func() {
		BeforeEach(func() {
			envoyConfParser.GetClustersCall.Returns.Clusters = testClusters()
			envoyConfParser.GetClustersCall.Returns.NameToPortMap = map[string]string{
				"0-service-cluster": "61001",
				"1-service-cluster": "61002",
				"2-service-cluster": "61003",
			}
		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		Describe("Good configuration", func() {
			BeforeEach(func() {
				var err error
				configFile, err = nginxConfig.Generate(envoyConfFile)
				Expect(err).ShouldNot(HaveOccurred())

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

					It("should specify the ssl certificate", func() {
						// TODO: test this separately for each server that is listening
						certPath := filepath.Join(tmpdir, "cert.pem")
						matcher := fmt.Sprintf(`[\r\n]\s*ssl_certificate\s*%s;`, convertToUnixPath(certPath))
						re := regexp.MustCompile(matcher)
						sslCertLine := re.Find(config)
						Expect(sslCertLine).NotTo(BeNil())
					})

					It("should specify the ssl private key", func() {
						keyPath := filepath.Join(tmpdir, "key.pem")
						matcher := fmt.Sprintf(`[\r\n]\s*ssl_certificate_key\s*%s;`, convertToUnixPath(keyPath))
						re := regexp.MustCompile(matcher)
						sslCertKeyLine := re.Find(config)
						Expect(sslCertKeyLine).NotTo(BeNil())
					})

					It("should verify the ssl client certificate", func() {
						Expect(string(config)).To(ContainSubstring("ssl_verify_client on"))
					})

					It("should include the ssl_client_certificate directive", func() {
						caPath := filepath.Join(tmpdir, "ca.pem")
						matcher := fmt.Sprintf(`[\r\n]\s*ssl_client_certificate\s*%s;`, convertToUnixPath(caPath))
						re := regexp.MustCompile(matcher)
						sslCACertLine := re.Find(config)
						Expect(sslCACertLine).NotTo(BeNil())
					})
				})
			})
		})

		Describe("Bad configuration", func() {
			BeforeEach(func() {
				var err error
				tmpdir, err = ioutil.TempDir("", "conf")
				Expect(err).ShouldNot(HaveOccurred())
				nginxConfig = parser.NewNginxConfig(envoyConfParser, sdsCredParser, sdsValidationParser, tmpdir)
			})

			Context("when a listener port is missing for a cluster name", func() {
				BeforeEach(func() {
					envoyConfParser.GetClustersCall.Returns.Clusters = []parser.Cluster{{Name: "banana"}}
					envoyConfParser.GetClustersCall.Returns.NameToPortMap = map[string]string{}
				})

				It("should return a custom error", func() {
					_, err := nginxConfig.Generate(envoyConfFile)
					Expect(err).To(MatchError("port is missing for cluster name banana"))
				})
			})

		})

		Context("when ioutil fails to write the envoy_nginx.conf", func() {
			BeforeEach(func() {
				nginxConfig = parser.NewNginxConfig(envoyConfParser, sdsCredParser, sdsValidationParser, "not-a-real-dir")
			})
			// We do not test that ioutil.WriteFile fails for cert/key because
			// our trick to cause that function to fail only works once!
			// The trick is to pass a directory that isn't real.
			It("returns a helpful error message", func() {
				_, err := nginxConfig.Generate(envoyConfFile)
				Expect(err.Error()).To(ContainSubstring("Failed to write envoy_nginx.conf:"))
			})
		})
	})
})

func convertToUnixPath(path string) string {
	path = strings.Replace(path, "C:", "", -1)
	path = strings.Replace(path, "\\", "/", -1)
	return path
}

func testClusters() []parser.Cluster {
	return []parser.Cluster{
		{
			Hosts: []parser.Host{
				{
					SocketAddress: parser.SocketAddress{
						Address:   "172.30.2.245",
						PortValue: "8080",
					},
				},
			},
			Name: "0-service-cluster",
		},
		{
			Hosts: []parser.Host{
				{
					SocketAddress: parser.SocketAddress{
						Address:   "172.30.2.245",
						PortValue: "2222",
					},
				},
			},
			Name: "1-service-cluster",
		},
		{
			Hosts: []parser.Host{
				{
					SocketAddress: parser.SocketAddress{
						Address:   "172.30.2.245",
						PortValue: "1234",
					},
				},
			},
			Name: "2-service-cluster",
		},
	}
}
