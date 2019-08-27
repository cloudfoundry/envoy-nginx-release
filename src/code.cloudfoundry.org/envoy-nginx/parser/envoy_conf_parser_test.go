package parser_test

import (
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/envoy-nginx/parser"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EnvoyConfigParser", func() {
	var (
		envoyConfParser parser.EnvoyConfParser
	)

	BeforeEach(func() {
		envoyConfParser = parser.NewEnvoyConfParser()
	})

	Describe("ReadUnmarshalEnvoyConfig", func() {
		It("returns an Envoy config file", func() {
			conf, err := envoyConfParser.ReadUnmarshalEnvoyConfig(EnvoyConfigFixture)
			Expect(err).NotTo(HaveOccurred())

			Expect(conf.StaticResources.Clusters).To(HaveLen(3))
			Expect(conf.StaticResources.Listeners).To(HaveLen(3))
		})

		Context("when envoyConf doesn't exist", func() {
			It("should return a read error", func() {
				_, err := envoyConfParser.ReadUnmarshalEnvoyConfig("not-a-real-file")
				Expect(err.Error()).To(ContainSubstring("Failed to read envoy config: open not-a-real-file:"))
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
					_, err := envoyConfParser.ReadUnmarshalEnvoyConfig(invalidYamlFile)
					Expect(err).To(MatchError("Failed to unmarshal envoy config: yaml: could not find expected directive name"))
				})
			})
		})
	})

	Describe("GetClusters", func() {
		Context("when the envoy conf file is successfully unmarshaled", func() {
			It("returns a set of clusters", func() {
				conf, err := envoyConfParser.ReadUnmarshalEnvoyConfig(EnvoyConfigFixture)
				Expect(err).NotTo(HaveOccurred())
				clusters, nameToPortAndCiphersMap := envoyConfParser.GetClusters(conf)

				Expect(clusters).To(HaveLen(3))
				Expect(nameToPortAndCiphersMap).To(HaveLen(3))

				Expect(nameToPortAndCiphersMap["0-service-cluster"].Ciphers).To(Equal("ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256"))
				Expect(nameToPortAndCiphersMap["1-service-cluster"].Ciphers).To(Equal("ECDHE-RSA-AES256-GCM-SHA384"))
				Expect(nameToPortAndCiphersMap["2-service-cluster"].Ciphers).To(Equal(""))
			})
		})
		Context("when envoyConf has no data", func() {
			It("returns empty clusters and nameToPortMap", func() {
				emptyConf, err := envoyConfParser.ReadUnmarshalEnvoyConfig("not-a-real-file")
				Expect(err.Error()).To(ContainSubstring("Failed to read envoy config: open not-a-real-file:"))
				clusters, nameToPortAndCiphersMap := envoyConfParser.GetClusters(emptyConf)
				Expect(clusters).To(HaveLen(0))
				Expect(nameToPortAndCiphersMap).To(HaveLen(0))
			})
		})
	})

	Describe("GetMTLS", func() {
		Context("when the envoy conf file is successfully unmarshaled", func() {
			It("returns whether MTLS is enabled", func() {
				conf, err := envoyConfParser.ReadUnmarshalEnvoyConfig(EnvoyConfigFixture)
				Expect(err).NotTo(HaveOccurred())
				mtls := envoyConfParser.GetMTLS(conf)
				Expect(mtls).To(BeTrue())
			})
		})

		Context("when require_client_certificate is false", func() {
			It("should return false", func() {
				envoyConfNoMTLS := "../fixtures/cf_assets_envoy_config/envoy-without-mtls.yaml"
				conf, err := envoyConfParser.ReadUnmarshalEnvoyConfig(envoyConfNoMTLS)
				Expect(err).NotTo(HaveOccurred())
				mtls := envoyConfParser.GetMTLS(conf)
				Expect(mtls).To(BeFalse())
			})
		})

		Context("when envoyConf has no data", func() {
			It("returns empty clusters and nameToPortMap", func() {
				emptyConf, err := envoyConfParser.ReadUnmarshalEnvoyConfig("not-a-real-file")
				Expect(err.Error()).To(ContainSubstring("Failed to read envoy config: open not-a-real-file:"))
				mtls := envoyConfParser.GetMTLS(emptyConf)
				Expect(mtls).To(BeFalse())
			})
		})

	})
})
