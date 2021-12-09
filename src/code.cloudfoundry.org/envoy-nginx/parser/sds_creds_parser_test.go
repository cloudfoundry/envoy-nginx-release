package parser_test

import (
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/envoy-nginx/parser"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SdsCredParser", func() {
	var (
		sdsCredParser parser.SdsCredParser
	)

	BeforeEach(func() {
		sdsCredsFile := "../fixtures/cf_assets_envoy_config/sds-id-cert-and-key.yaml"
		sdsCredParser = parser.NewSdsIdCredParser(sdsCredsFile)
	})

	Describe("GetCertAndKey", func() {
		It("reads the sds file and returns the cert and key", func() {
			cert, key, err := sdsCredParser.GetCertAndKey()
			Expect(err).NotTo(HaveOccurred())
			Expect(cert).To(ContainSubstring("-----BEGIN CERTIFICATE-----"))
			Expect(key).To(ContainSubstring("-----BEGIN RSA PRIVATE KEY-----"))
		})

		Context("when the resources section is not found", func() {
			var invalidSdsFile string

			BeforeEach(func() {
				tmpFile, err := ioutil.TempFile(os.TempDir(), "invalid-sds.yaml")
				Expect(err).NotTo(HaveOccurred())
				_, err = tmpFile.Write([]byte(""))
				Expect(err).NotTo(HaveOccurred())

				invalidSdsFile = tmpFile.Name()
				sdsCredParser = parser.NewSdsIdCredParser(invalidSdsFile)
			})

			AfterEach(func() {
				os.Remove(invalidSdsFile)
			})

			It("returns a helpful error", func() {
				_, _, err := sdsCredParser.GetCertAndKey()
				Expect(err).To(MatchError("resources section not found in sds cred file"))
			})
		})

		Context("when sdsCreds doesn't exist", func() {
			BeforeEach(func() {
				sdsCredParser = parser.NewSdsIdCredParser("not-a-real-file")
			})

			It("should return a read error", func() {
				_, _, err := sdsCredParser.GetCertAndKey()
				Expect(err.Error()).To(ContainSubstring("Failed to read sds creds: open not-a-real-file:"))
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
				sdsCredParser = parser.NewSdsIdCredParser(invalidYamlFile)
			})

			AfterEach(func() {
				os.Remove(invalidYamlFile)
			})

			It("should return unmarshal error", func() {
				_, _, err := sdsCredParser.GetCertAndKey()
				Expect(err).To(MatchError("Failed to unmarshal sds creds: yaml: could not find expected directive name"))
			})
		})
	})
})
