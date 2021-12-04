package parser_test

import (
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/envoy-nginx/parser"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SdsIdValidationParser", func() {
	var (
		sdsIdValidationParser parser.SdsIdValidationParser
	)

	BeforeEach(func() {
		sdsIdValidationFile := "../fixtures/cf_assets_envoy_config/sds-id-validation-context.yaml"
		sdsIdValidationParser = parser.NewSdsIdValidationParser(sdsIdValidationFile)
	})

	Describe("GetCACert", func() {
		It("reads the sds file and returns the trusted ca cert", func() {
			cert, err := sdsIdValidationParser.GetCACert()
			Expect(err).NotTo(HaveOccurred())
			Expect(cert).To(ContainSubstring("-----BEGIN CERTIFICATE-----"))
		})

		Context("when the resources section is not found", func() {
			var invalidSdsFile string

			BeforeEach(func() {
				tmpFile, err := ioutil.TempFile(os.TempDir(), "invalid-sds-server-validation-context.yaml")
				Expect(err).NotTo(HaveOccurred())
				_, err = tmpFile.Write([]byte(""))
				Expect(err).NotTo(HaveOccurred())

				invalidSdsFile = tmpFile.Name()
				sdsIdValidationParser = parser.NewSdsIdValidationParser(invalidSdsFile)
			})

			AfterEach(func() {
				os.Remove(invalidSdsFile)
			})

			It("returns a helpful error", func() {
				_, err := sdsIdValidationParser.GetCACert()
				Expect(err).To(MatchError("resources section not found in sds-server-validation-context.yaml"))
			})
		})

		Context("when sdsCreds doesn't exist", func() {
			BeforeEach(func() {
				sdsIdValidationParser = parser.NewSdsIdValidationParser("not-a-real-file")
			})
			It("should return a read error", func() {
				_, err := sdsIdValidationParser.GetCACert()
				Expect(err.Error()).To(ContainSubstring("Failed to read sds server validation context:"))
			})
		})

		Context("when the config contains invalid yaml", func() {
			var invalidYamlFile string

			BeforeEach(func() {
				tmpFile, err := ioutil.TempFile(os.TempDir(), "invalid.yaml")
				Expect(err).NotTo(HaveOccurred())
				_, err = tmpFile.Write([]byte("%%%"))
				Expect(err).NotTo(HaveOccurred())

				invalidYamlFile = tmpFile.Name()
				sdsIdValidationParser = parser.NewSdsIdValidationParser(invalidYamlFile)
			})

			AfterEach(func() {
				os.Remove(invalidYamlFile)
			})

			It("should return unmarshal error", func() {
				_, err := sdsIdValidationParser.GetCACert()
				Expect(err.Error()).To(ContainSubstring("Failed to unmarshal sds server validation context: yaml: could not find expected directive name"))
			})
		})
	})
})
