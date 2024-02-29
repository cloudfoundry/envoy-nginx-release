package parser_test

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
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
		sdsIdCredParser     *fakes.SdsCredParser
		sdsC2CCredParser    *fakes.SdsCredParser
		sdsValidationParser *fakes.SdsIdValidationParser
	)

	BeforeEach(func() {
		sdsIdCredParser = &fakes.SdsCredParser{}
		sdsC2CCredParser = &fakes.SdsCredParser{}
		sdsValidationParser = &fakes.SdsIdValidationParser{}
		envoyConfParser = &fakes.EnvoyConfParser{}

		var err error
		tmpdir, err = os.MkdirTemp("", "nginx")
		Expect(err).ShouldNot(HaveOccurred())
		err = os.Mkdir(filepath.Join(tmpdir, "conf"), os.ModePerm)
		Expect(err).ShouldNot(HaveOccurred())

		nginxConfig = parser.NewNginxConfig(envoyConfParser, []parser.SdsCredParser{sdsIdCredParser, sdsC2CCredParser}, sdsValidationParser, tmpdir)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpdir)).NotTo(HaveOccurred())
	})

	Describe("WriteTLSFiles", func() {
		BeforeEach(func() {
			sdsIdCredParser.GetCertAndKeyCall.Returns.Cert = "some-id-cert"
			sdsIdCredParser.GetCertAndKeyCall.Returns.Key = "some-id-key"
			sdsIdCredParser.ConfigTypeCall.Returns.ConfigType = parser.SdsIdConfigType
			sdsC2CCredParser.GetCertAndKeyCall.Returns.Cert = "some-c2c-cert"
			sdsC2CCredParser.GetCertAndKeyCall.Returns.Key = "some-c2c-key"
			sdsC2CCredParser.ConfigTypeCall.Returns.ConfigType = parser.SdsC2CConfigType
			sdsValidationParser.GetCACertCall.Returns.CA = "some-ca-cert"
		})

		It("should have written cert, key, and ca", func() {
			err := nginxConfig.WriteTLSFiles()
			Expect(err).ShouldNot(HaveOccurred())

			certPath := filepath.Join(tmpdir, "id-cert.pem")
			keyPath := filepath.Join(tmpdir, "id-key.pem")

			cert, err := os.ReadFile(string(certPath))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(cert)).To(Equal("some-id-cert"))

			key, err := os.ReadFile(string(keyPath))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(key)).To(Equal("some-id-key"))

			certPath = filepath.Join(tmpdir, "c2c-cert.pem")
			keyPath = filepath.Join(tmpdir, "c2c-key.pem")

			cert, err = os.ReadFile(string(certPath))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(cert)).To(Equal("some-c2c-cert"))

			key, err = os.ReadFile(string(keyPath))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(key)).To(Equal("some-c2c-key"))

			caPath := filepath.Join(tmpdir, "id-ca.pem")
			ca, err := os.ReadFile(string(caPath))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(ca)).To(Equal("some-ca-cert"))
		})

		Context("when c2c sds cred parser is not provided", func() {
			BeforeEach(func() {
				nginxConfig = parser.NewNginxConfig(envoyConfParser, []parser.SdsCredParser{sdsIdCredParser}, sdsValidationParser, tmpdir)
			})

			It("only writes cert and key for provided parsers", func() {
				err := nginxConfig.WriteTLSFiles()
				Expect(err).ShouldNot(HaveOccurred())

				certPath := filepath.Join(tmpdir, "id-cert.pem")
				keyPath := filepath.Join(tmpdir, "id-key.pem")

				cert, err := os.ReadFile(string(certPath))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(string(cert)).To(Equal("some-id-cert"))

				key, err := os.ReadFile(string(keyPath))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(string(key)).To(Equal("some-id-key"))

				certPath = filepath.Join(tmpdir, "c2c-cert.pem")
				keyPath = filepath.Join(tmpdir, "c2c-key.pem")

				_, err = os.Stat(string(certPath))
				Expect(err).Should(HaveOccurred())

				_, err = os.Stat(string(keyPath))
				Expect(err).Should(HaveOccurred())

				caPath := filepath.Join(tmpdir, "id-ca.pem")
				ca, err := os.ReadFile(string(caPath))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(string(ca)).To(Equal("some-ca-cert"))
			})
		})

		Context("when id sds cred parser fails to get cert and key", func() {
			BeforeEach(func() {
				sdsIdCredParser.GetCertAndKeyCall.Returns.Error = errors.New("banana")
			})

			It("returns a helpful error message", func() {
				err := nginxConfig.WriteTLSFiles()
				Expect(err).To(MatchError("get cert and key from sds cred parser: banana"))
			})
		})

		Context("when c2c sds cred parser fails to get cert and key", func() {
			BeforeEach(func() {
				sdsC2CCredParser.GetCertAndKeyCall.Returns.Error = errors.New("banana")
			})

			It("returns a helpful error message", func() {
				err := nginxConfig.WriteTLSFiles()
				Expect(err).To(MatchError("get cert and key from sds cred parser: banana"))
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
			envoyConfParser.GetClustersCall.Returns.NameToListeners = map[string][]parser.ListenerInfo{
				"service-cluster-8080": {
					{Port: "61001", MTLS: true, Ciphers: "ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256", SdsConfigType: parser.SdsIdConfigType},
				},
				"service-cluster-2222": {
					{Port: "61002", MTLS: true, Ciphers: "banana_ciphers", SdsConfigType: parser.SdsIdConfigType},
				},
				"service-cluster-1234": {
					{Port: "61003", MTLS: true, Ciphers: "", SdsConfigType: parser.SdsIdConfigType},
					{Port: "61004", MTLS: false, Ciphers: "", SdsConfigType: parser.SdsC2CConfigType},
				},
			}
		})

		Context("when envoyConf and sdsCreds files are configured correctly", func() {
			It("should generate a valid nginx.conf", func() {
				var err error
				err = nginxConfig.Generate(EnvoyConfigFixture)
				Expect(err).ShouldNot(HaveOccurred())

				config, err = os.ReadFile(nginxConfig.GetConfFile())
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

				clusterToListeners := parseNginxConfig(config)

				expectedClientCertPath := convertToUnixPath(filepath.Join(tmpdir, "id-ca.pem"))

				By("having valid listeners for server 8080", func() {
					Expect(clusterToListeners["service-cluster-8080"]).To(Equal([]listenerInfo{
						{
							Port:           61001,
							ClientVerify:   true,
							ClientCertPath: expectedClientCertPath,
							Ciphers:        "ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256",
							Cert:           convertToUnixPath(filepath.Join(tmpdir, "id-cert.pem")),
							Key:            convertToUnixPath(filepath.Join(tmpdir, "id-key.pem")),
						},
					}))
				})
				By("having valid listeners for server 2222", func() {
					Expect(clusterToListeners["service-cluster-2222"]).To(Equal([]listenerInfo{
						{
							Port:           61002,
							ClientVerify:   true,
							ClientCertPath: expectedClientCertPath,
							Ciphers:        "banana_ciphers",
							Cert:           convertToUnixPath(filepath.Join(tmpdir, "id-cert.pem")),
							Key:            convertToUnixPath(filepath.Join(tmpdir, "id-key.pem")),
						},
					}))
				})
				By("having valid listeners for server 1234", func() {
					Expect(clusterToListeners["service-cluster-1234"]).To(Equal([]listenerInfo{
						{
							Port:           61003,
							ClientVerify:   true,
							ClientCertPath: expectedClientCertPath,
							Cert:           convertToUnixPath(filepath.Join(tmpdir, "id-cert.pem")),
							Key:            convertToUnixPath(filepath.Join(tmpdir, "id-key.pem")),
						},
						{
							Port:         61004,
							ClientVerify: false,
							Cert:         convertToUnixPath(filepath.Join(tmpdir, "c2c-cert.pem")),
							Key:          convertToUnixPath(filepath.Join(tmpdir, "c2c-key.pem")),
						},
					}))
				})
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

		Context("when ioutil fails to write the nginx.conf", func() {
			BeforeEach(func() {
				nginxConfig = parser.NewNginxConfig(envoyConfParser, []parser.SdsCredParser{sdsIdCredParser, sdsC2CCredParser}, sdsValidationParser, "not-a-real-dir")
			})
			// We do not test that os.WriteFile fails for cert/key because
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

type listenerInfo struct {
	Port           int
	ClientVerify   bool
	ClientCertPath string
	Ciphers        string
	Cert           string
	Key            string
}

func parseNginxConfig(config []byte) map[string][]listenerInfo {
	serverConfigs := strings.Split(string(config), "server {")
	portsToClientCertificateMap := map[string][]listenerInfo{}
	for i := 1; i < len(serverConfigs); i++ {
		serverConfig := serverConfigs[i]

		re := regexp.MustCompile(`proxy_pass\s*(.*);`)
		matches := re.FindStringSubmatch(serverConfig)
		Expect(matches).To(HaveLen(2))
		name := matches[1]

		re = regexp.MustCompile(`listen (.*) ssl;`)
		matches = re.FindStringSubmatch(serverConfig)
		Expect(matches).To(HaveLen(2))
		portStr := matches[1]
		port, err := strconv.Atoi(portStr)
		Expect(err).NotTo(HaveOccurred())

		re = regexp.MustCompile(`ssl_verify_client\s*(.*);"`)
		matches = re.FindStringSubmatch(serverConfig)
		var verify bool
		if len(matches) > 0 && matches[1] == "on" {
			verify = true
		}

		re = regexp.MustCompile(`ssl_client_certificate\s*(.*);`)
		matches = re.FindStringSubmatch(serverConfig)
		var certPath string
		if len(matches) > 1 {
			certPath = matches[1]
		}

		re = regexp.MustCompile(`ssl_ciphers\s*(.*);`)
		matches = re.FindStringSubmatch(serverConfig)
		var ciphers string
		if len(matches) > 1 {
			ciphers = matches[1]
		}

		re = regexp.MustCompile(`ssl_certificate\s*(.*);`)
		matches = re.FindStringSubmatch(serverConfig)
		var cert string
		if len(matches) > 1 {
			cert = matches[1]
		}

		re = regexp.MustCompile(`ssl_certificate_key\s*(.*);`)
		matches = re.FindStringSubmatch(serverConfig)
		var key string
		if len(matches) > 1 {
			key = matches[1]
		}

		portsToClientCertificateMap[name] = append(portsToClientCertificateMap[name], listenerInfo{
			Port:           port,
			ClientVerify:   verify,
			ClientCertPath: certPath,
			Ciphers:        ciphers,
			Cert:           cert,
			Key:            key,
		})
	}
	return portsToClientCertificateMap
}

func testClusters() []parser.Cluster {
	return []parser.Cluster{
		{
			Name: "service-cluster-8080",
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
			Name: "service-cluster-2222",
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
			Name: "service-cluster-1234",
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
