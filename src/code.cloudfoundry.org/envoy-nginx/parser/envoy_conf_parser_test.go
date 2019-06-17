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
		envoyConfFile   string
		envoyConfParser parser.EnvoyConfParser
	)

	BeforeEach(func() {
		envoyConfFile = "../fixtures/cf_assets_envoy_config/envoy.yaml"
		envoyConfParser = parser.NewEnvoyConfParser()
	})

	Describe("GetClusters", func() {
		It("returns a set of clusters", func() {
			clusters, nameToPortMap, err := envoyConfParser.GetClusters(envoyConfFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(clusters).To(HaveLen(3))
			Expect(nameToPortMap).To(HaveLen(3))
		})

		Context("when envoyConf doesn't exist", func() {
			It("should return a read error", func() {
				_, _, err := envoyConfParser.GetClusters("not-a-real-file")
				Expect(err).To(MatchError("Failed to read envoy config: open not-a-real-file: no such file or directory"))
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
					_, _, err := envoyConfParser.GetClusters(invalidYamlFile)
					Expect(err).To(MatchError("Failed to unmarshal envoy conf: yaml: could not find expected directive name"))
				})
			})
		})
	})
})
