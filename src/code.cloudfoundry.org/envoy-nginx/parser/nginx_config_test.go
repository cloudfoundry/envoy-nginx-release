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
		tmpdir string
		config []byte

		envoyConfParser     *fakes.EnvoyConfParser
		nginxConfig         parser.NginxConfig
		sdsCredParser       *fakes.SdsCredParser
		sdsValidationParser *fakes.SdsServerValidationParser
	)

	BeforeEach(func() {
		sdsCredParser = &fakes.SdsCredParser{}
		sdsValidationParser = &fakes.SdsServerValidationParser{}
		envoyConfParser = &fakes.EnvoyConfParser{}

		var err error
		tmpdir, err = ioutil.TempDir("", "nginx")
		Expect(err).ShouldNot(HaveOccurred())
		err = os.Mkdir(filepath.Join(tmpdir, "conf"), os.ModePerm)
		Expect(err).ShouldNot(HaveOccurred())

		nginxConfig = parser.NewNginxConfig(envoyConfParser, sdsCredParser, sdsValidationParser, tmpdir)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpdir)).NotTo(HaveOccurred())
	})

	Describe("WriteTLSFiles", func() {
		BeforeEach(func() {
			sdsCredParser.GetCertAndKeyCall.Returns.Cert = "some-cert"
			sdsCredParser.GetCertAndKeyCall.Returns.Key = "some-key"
			sdsValidationParser.GetCACertCall.Returns.CA = "some-ca-cert"
		})

		It("should have written cert, key, and ca", func() {
			err := nginxConfig.WriteTLSFiles()
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
				err := nginxConfig.WriteTLSFiles()
				Expect(err).To(MatchError("get cert and key from sds server cred parser: banana"))
			})
		})

		Context("when sds validation context parser fails to get ca", func() {
			BeforeEach(func() {
				sdsValidationParser.GetCACertCall.Returns.Error = errors.New("banana")
			})

			It("returns a helpful error message", func() {
				err := nginxConfig.WriteTLSFiles()
				Expect(err).To(MatchError("get ca cert from sds server validation parser: banana"))
			})
		})

		Context("when sds validation context parser returns an empty ca", func() {
			BeforeEach(func() {
				sdsValidationParser.GetCACertCall.Returns.CA = ""
			})

			It("does not create a ca.pem", func() {
				err := nginxConfig.WriteTLSFiles()
				Expect(err).ShouldNot(HaveOccurred())

				caPath := filepath.Join(tmpdir, "ca.pem")
				_, err = os.Stat(caPath)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Generate", func() {
		BeforeEach(func() {
			envoyConfParser.ReadUnmarshalEnvoyConfigCall.Returns.Error = nil
			envoyConfParser.GetClustersCall.Returns.Clusters = testClusters()
			envoyConfParser.GetClustersCall.Returns.NameToPortAndCiphersMap = map[string]parser.PortAndCiphers{
				"0-service-cluster": parser.PortAndCiphers{"61001", "ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256"},
				"1-service-cluster": parser.PortAndCiphers{"61002", "banana_ciphers"},
				"2-service-cluster": parser.PortAndCiphers{"61003", ""},
			}
			envoyConfParser.GetMTLSCall.Returns.MTLS = true
		})

		Context("when envoyConf and sdsCreds files are configured correctly", func() {
			It("should generate a valid nginx.conf", func() {
				var err error
				err = nginxConfig.Generate(EnvoyConfigFixture)
				Expect(err).ShouldNot(HaveOccurred())

				config, err = ioutil.ReadFile(nginxConfig.GetConfFile())
				Expect(err).ShouldNot(HaveOccurred())

				By("having a valid pid directive", func() {
					// e.g. pid /Temp/nginx_024.pid;
					re := regexp.MustCompile(`[\r\n]pid\s*[\w/.]+;`)
					match := re.Find(config)
					Expect(match).NotTo(BeNil())
				})

				By("having an upstream server with addr 172.30.2.245:8080", func() {
					re := regexp.MustCompile(`[\r\n]\s*server\s*172.30.2.245:8080;`)
					match := re.Find(config)
					Expect(match).NotTo(BeNil())
				})

				By("having an upstream server with addr 172.30.2.245:2222", func() {
					re := regexp.MustCompile(`[\r\n]\s*server\s*172.30.2.245:2222;`)
					match := re.Find(config)
					Expect(match).NotTo(BeNil())
				})

				By("having an upstream server with addr 172.30.2.245:1234", func() {
					re := regexp.MustCompile(`[\r\n]\s*server\s*172.30.2.245:1234;`)
					match := re.Find(config)
					Expect(match).NotTo(BeNil())
				})

				By("having a valid server listening on 61001", func() {
					re := regexp.MustCompile(`[\r\n]\s*listen\s*61001\s*ssl;`)
					match := re.Find(config)
					Expect(match).NotTo(BeNil())

					re = regexp.MustCompile(`[\r\n]\s*proxy_pass\s*0-service-cluster;`)
					match = re.Find(config)
					Expect(match).NotTo(BeNil())
				})

				By("having a valid server listening on 61002", func() {
					re := regexp.MustCompile(`[\r\n]\s*listen\s*61002\s*ssl;`)
					match := re.Find(config)
					Expect(match).NotTo(BeNil())

					re = regexp.MustCompile(`[\r\n]\s*proxy_pass\s*1-service-cluster;`)
					match = re.Find(config)
					Expect(match).NotTo(BeNil())
				})

				By("having a valid server listening on 61003", func() {
					re := regexp.MustCompile(`[\r\n]\s*listen\s*61003\s*ssl;`)
					match := re.Find(config)
					Expect(match).NotTo(BeNil())

					re = regexp.MustCompile(`[\r\n]\s*proxy_pass\s*2-service-cluster;`)
					match = re.Find(config)
					Expect(match).NotTo(BeNil())
				})

				By("specifying the ssl certificate", func() {
					// TODO: test this separately for each server that is listening
					certPath := filepath.Join(tmpdir, "cert.pem")
					matcher := fmt.Sprintf(`[\r\n]\s*ssl_certificate\s*%s;`, convertToUnixPath(certPath))
					re := regexp.MustCompile(matcher)
					sslCertLine := re.Find(config)
					Expect(sslCertLine).NotTo(BeNil())
				})

				By("specifying the ssl private key", func() {
					keyPath := filepath.Join(tmpdir, "key.pem")
					matcher := fmt.Sprintf(`[\r\n]\s*ssl_certificate_key\s*%s;`, convertToUnixPath(keyPath))
					re := regexp.MustCompile(matcher)
					sslCertKeyLine := re.Find(config)
					Expect(sslCertKeyLine).NotTo(BeNil())
				})

				By("verifying the ssl client certificate", func() {
					Expect(string(config)).To(ContainSubstring("ssl_verify_client on"))
				})

				By("including the ssl_client_certificate directive", func() {
					caPath := filepath.Join(tmpdir, "ca.pem")
					matcher := fmt.Sprintf(`[\r\n]\s*ssl_client_certificate\s*%s;`, convertToUnixPath(caPath))
					re := regexp.MustCompile(matcher)
					sslCACertLine := re.Find(config)
					Expect(sslCACertLine).NotTo(BeNil())
				})

				By("having a ssl_prefer_server_ciphers directive", func() {
					Expect(string(config)).To(ContainSubstring("ssl_prefer_server_ciphers on"))
				})

				By("having a ssl_ciphers directive", func() {
					Expect(string(config)).To(ContainSubstring("ssl_ciphers ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256"))
					Expect(string(config)).To(ContainSubstring("ssl_ciphers banana_ciphers"))
					Expect(string(config)).To(ContainSubstring("ssl_ciphers ;"))
				})
			})
		})

		Context("when trusted ca certificates are not provided", func() {
			BeforeEach(func() {
				envoyConfParser.GetMTLSCall.Returns.MTLS = false
				err := nginxConfig.Generate(EnvoyConfigFixture)
				Expect(err).ShouldNot(HaveOccurred())

				config, err = ioutil.ReadFile(nginxConfig.GetConfFile())
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("disables mtls", func() {
				Expect(envoyConfParser.GetMTLSCall.CallCount).To(Equal(1))
				By("not verifying the ssl client certificate")
				Expect(string(config)).NotTo(ContainSubstring("ssl_verify_client on"))

				By("not including the ssl_client_certificate directive")
				caPath := filepath.Join(tmpdir, "ca.pem")
				matcher := fmt.Sprintf(`[\r\n]\s*ssl_client_certificate\s*%s;`, convertToUnixPath(caPath))
				re := regexp.MustCompile(matcher)
				sslCACertLine := re.Find(config)
				Expect(sslCACertLine).To(BeNil())
			})
		})

		Context("when ReadUnmarshalEnvoyConfig fails", func() {
			BeforeEach(func() {
				envoyConfParser.ReadUnmarshalEnvoyConfigCall.Returns.Error = errors.New("banana")
			})
			It("should return a custom error", func() {
				err := nginxConfig.Generate(EnvoyConfigFixture)
				Expect(err).To(MatchError("read and unmarshal Envoy config: banana"))
			})
		})

		Context("when a listener port is missing for a cluster name", func() {
			BeforeEach(func() {
				envoyConfParser.GetClustersCall.Returns.Clusters = []parser.Cluster{{Name: "banana"}}
				envoyConfParser.GetClustersCall.Returns.NameToPortAndCiphersMap = map[string]parser.PortAndCiphers{}
			})

			It("should return a custom error", func() {
				err := nginxConfig.Generate(EnvoyConfigFixture)
				Expect(err).To(MatchError("port is missing for cluster name banana"))
			})
		})

		Context("when ioutil fails to write the nginx.conf", func() {
			BeforeEach(func() {
				nginxConfig = parser.NewNginxConfig(envoyConfParser, sdsCredParser, sdsValidationParser, "not-a-real-dir")
			})
			// We do not test that ioutil.WriteFile fails for cert/key because
			// our trick to cause that function to fail only works once!
			// The trick is to pass a directory that isn't real.
			It("returns a helpful error message", func() {
				err := nginxConfig.Generate(EnvoyConfigFixture)
				Expect(err.Error()).To(ContainSubstring("nginx.conf - write file failed:"))
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
			Name: "0-service-cluster",
			LoadAssignment: parser.LoadAssignment{
				Endpoints: []parser.Endpoints{
					{
						LBEndpoints: []parser.LBEndpoints{
							{
								Endpoint: parser.Endpoint{
									Address: parser.Address{
										SocketAddress: parser.SocketAddress{
											Address:   "172.30.2.245",
											PortValue: "8080",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "1-service-cluster",
			LoadAssignment: parser.LoadAssignment{
				Endpoints: []parser.Endpoints{
					{
						LBEndpoints: []parser.LBEndpoints{
							{
								Endpoint: parser.Endpoint{
									Address: parser.Address{
										SocketAddress: parser.SocketAddress{
											Address:   "172.30.2.245",
											PortValue: "2222",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "2-service-cluster",
			LoadAssignment: parser.LoadAssignment{
				Endpoints: []parser.Endpoints{
					{
						LBEndpoints: []parser.LBEndpoints{
							{
								Endpoint: parser.Endpoint{
									Address: parser.Address{
										SocketAddress: parser.SocketAddress{
											Address:   "172.30.2.245",
											PortValue: "1234",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
