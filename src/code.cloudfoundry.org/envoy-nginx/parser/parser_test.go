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

var _ = Describe("Parser", func() {
	var (
		envoyConfFile   string
		sdsCredsFile    string
		tmpdir          string
		configFile      string
		config          []byte
		envoyConfParser *fakes.EnvoyConfParser
		p               parser.Parser
		sdsCredParser   *fakes.SdsCredParser
	)

	Describe("GenerateConf", func() {
		BeforeEach(func() {
			envoyConfFile = "../fixtures/cf_assets_envoy_config/envoy.yaml"
			sdsCredsFile = "../fixtures/cf_assets_envoy_config/sds-server-cert-and-key.yaml"

			var err error
			tmpdir, err = ioutil.TempDir("", "conf")
			Expect(err).ShouldNot(HaveOccurred())

			sdsCredParser = &fakes.SdsCredParser{}
			sdsCredParser.GetCertAndKeyCall.Returns.Cert = "some-cert"
			sdsCredParser.GetCertAndKeyCall.Returns.Key = "some-key"

			envoyConfParser = &fakes.EnvoyConfParser{}
			envoyConfParser.GetClustersCall.Returns.Clusters = []parser.Cluster{
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
			envoyConfParser.GetClustersCall.Returns.NameToPortMap = map[string]string{
				"0-service-cluster": "61001",
				"1-service-cluster": "61002",
				"2-service-cluster": "61003",
			}

			p = parser.NewParser(envoyConfParser, sdsCredParser)
		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		Describe("Good configuration", func() {
			BeforeEach(func() {
				err := p.GenerateConf(envoyConfFile, sdsCredsFile, tmpdir)
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

					It("should specify the ssl certificate", func() {
						// TODO: test this separately for each server that is listening
						certPath := filepath.Join(tmpdir, "cert.pem")
						matcher := fmt.Sprintf(`[\r\n]\s*ssl_certificate\s*%s;`, convertToUnixPath(certPath))
						re := regexp.MustCompile(matcher)
						sslCertLine := re.Find(config)
						Expect(sslCertLine).NotTo(BeNil())

						sslCertPath := filepath.Join(tmpdir, "cert.pem")
						sslCert, err := ioutil.ReadFile(string(sslCertPath))
						Expect(err).ShouldNot(HaveOccurred())

						Expect(string(sslCert)).To(Equal("some-cert"))
					})

					It("should specify the ssl private key", func() {
						keyPath := filepath.Join(tmpdir, "key.pem")
						matcher := fmt.Sprintf(`[\r\n]\s*ssl_certificate_key\s*%s;`, convertToUnixPath(keyPath))
						re := regexp.MustCompile(matcher)
						sslCertKeyLine := re.Find(config)
						Expect(sslCertKeyLine).NotTo(BeNil())

						sslCertKeyPath := filepath.Join(tmpdir, "key.pem")
						sslCertKey, err := ioutil.ReadFile(string(sslCertKeyPath))
						Expect(err).ShouldNot(HaveOccurred())

						Expect(string(sslCertKey)).To(Equal("some-key"))
					})
				})
			})
		})

		Describe("Bad configuration", func() {
			Context("when a listener port is missing for a cluster name", func() {
				BeforeEach(func() {
					envoyConfParser.GetClustersCall.Returns.Clusters = []parser.Cluster{{Name: "banana"}}
					envoyConfParser.GetClustersCall.Returns.NameToPortMap = map[string]string{}
				})

				It("should return a custom error", func() {
					err := p.GenerateConf(envoyConfFile, sdsCredsFile, tmpdir)
					Expect(err).To(MatchError("port is missing for cluster name banana"))
				})
			})

			Context("when sds cred parser fails to get cert and key", func() {
				BeforeEach(func() {
					sdsCredParser.GetCertAndKeyCall.Returns.Error = errors.New("banana")
				})

				It("returns a helpful error message", func() {
					err := p.GenerateConf(envoyConfFile, sdsCredsFile, tmpdir)
					Expect(err).To(MatchError("Failed to get cert and key from sds file: banana"))
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
