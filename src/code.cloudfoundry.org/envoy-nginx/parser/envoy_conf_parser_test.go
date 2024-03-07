package parser_test

import (
	"os"

	"code.cloudfoundry.org/envoy-nginx/parser"
	. "github.com/onsi/ginkgo/v2"
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
		It("returns an Envoy config file with the correct number of clusters and listeners", func() {
			conf, err := envoyConfParser.ReadUnmarshalEnvoyConfig(EnvoyConfigFixture)
			Expect(err).NotTo(HaveOccurred())

			Expect(conf.StaticResources.Clusters).To(HaveLen(2))
			Expect(conf.StaticResources.Listeners).To(HaveLen(3))

			conf, err = envoyConfParser.ReadUnmarshalEnvoyConfig(EnvoyOneListenerPerServerConfigFixture)
			Expect(err).NotTo(HaveOccurred())

			Expect(conf.StaticResources.Clusters).To(HaveLen(2))
			Expect(conf.StaticResources.Listeners).To(HaveLen(2))
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
				tmpFile, err := os.CreateTemp(os.TempDir(), "envoy-invalid.yaml")
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
			It("returns a set of clusters for config with multiple listeners per server", func() {
				conf, err := envoyConfParser.ReadUnmarshalEnvoyConfig(EnvoyConfigFixture)
				Expect(err).NotTo(HaveOccurred())
				clusters, nameToListeners := envoyConfParser.GetClusters(conf)

				Expect(clusters).To(HaveLen(2))
				Expect(nameToListeners).To(HaveLen(2))

				Expect(nameToListeners["service-cluster-8080"]).To(HaveLen(2))
				Expect(nameToListeners["service-cluster-8080"]).To(Equal([]parser.ListenerInfo{
					{Port: "61001", MTLS: true, Ciphers: "ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256", SdsConfigType: parser.SdsIdConfigType},
					{Port: "61443", MTLS: false, Ciphers: "ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256", SdsConfigType: parser.SdsC2CConfigType},
				}))
				Expect(nameToListeners["service-cluster-2222"]).To(Equal([]parser.ListenerInfo{
					{Port: "61002", MTLS: true, Ciphers: "ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256", SdsConfigType: parser.SdsIdConfigType},
				}))
			})

			It("returns a set of clusters for config with one listeners per server", func() {
				conf, err := envoyConfParser.ReadUnmarshalEnvoyConfig(EnvoyOneListenerPerServerConfigFixture)
				Expect(err).NotTo(HaveOccurred())
				clusters, nameToListeners := envoyConfParser.GetClusters(conf)

				Expect(clusters).To(HaveLen(2))
				Expect(nameToListeners).To(HaveLen(2))

				Expect(nameToListeners["0-service-cluster"]).To(Equal([]parser.ListenerInfo{
					{Port: "61001", MTLS: true, Ciphers: "ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256", SdsConfigType: parser.SdsIdConfigType},
				}))
				Expect(nameToListeners["1-service-cluster"]).To(Equal([]parser.ListenerInfo{
					{Port: "61002", MTLS: true, Ciphers: "ECDHE-RSA-AES256-GCM-SHA384", SdsConfigType: parser.SdsIdConfigType},
				}))
			})
		})

		Context("when envoyConf has no data", func() {
			It("returns empty clusters and nameToListeners", func() {
				emptyConf, err := envoyConfParser.ReadUnmarshalEnvoyConfig("not-a-real-file")
				Expect(err.Error()).To(ContainSubstring("Failed to read envoy config: open not-a-real-file:"))
				clusters, nameToListeners := envoyConfParser.GetClusters(emptyConf)
				Expect(clusters).To(HaveLen(0))
				Expect(nameToListeners).To(HaveLen(0))
			})
		})
	})
})
