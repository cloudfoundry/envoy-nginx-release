package parser_test

import (
	"io/ioutil"
	"os"
	"regexp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "code.cloudfoundry.org/envoy-nginx/parser"
)

var _ = Describe("Envoy-Nginx", func() {
	var configFile string
	var config []byte
	var err error

	BeforeEach(func() {
		os.Setenv("SDSCredsFile", "../test_config/cf_assets_envoy_config/sds-server-cert-and-key.yaml")
		configFile, err = GenerateConf()
		Expect(err).ShouldNot(HaveOccurred())
		config, err = ioutil.ReadFile(configFile)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Context("Generate nginx.conf", func() {
		It("should be a valid file of non-zero size", func() {
			f, err := os.Stat(configFile)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(f.Size()).Should(BeNumerically(">", 0))
		})

		Context("should have valid contents", func() {
			It("should have a valid pid directive", func() {
				// e.g. pid /Temp/nginx_024.pid;
				re := regexp.MustCompile(`[\r\n]pid\s*[\w/.]+;`)
				match := re.Find(config)
				Expect(match).NotTo(BeNil())
			})

			Context("app upstream", func() {
				It("should have an upstream server with addr 127.0.0.1:8080", func() {
					re := regexp.MustCompile(`[\r\n]\s*server\s*127.0.0.1:8080;`)
					match := re.Find(config)
					Expect(match).NotTo(BeNil())
				})
			})

			Context("sshd upstream", func() {
				It("should have an upstream server with addr 127.0.0.1:2222", func() {
					re := regexp.MustCompile(`[\r\n]\s*server\s*127.0.0.1:2222;`)
					match := re.Find(config)
					Expect(match).NotTo(BeNil())
				})
			})

			Context("server listening on 61001", func() {
				It("should have a listen 61001 ssl directive", func() {
					re := regexp.MustCompile(`[\r\n]\s*listen\s*61001\s*ssl;`)
					match := re.Find(config)
					Expect(match).NotTo(BeNil())
				})

				It("should have a proxy_pass directive to app", func() {
					re := regexp.MustCompile(`[\r\n]\s*proxy_pass\s*app;`)
					match := re.Find(config)
					Expect(match).NotTo(BeNil())
				})

				It("should have a valid ssl_certificate directive", func() {
					re := regexp.MustCompile(`[\r\n]\s*ssl_certificate\s*/tmp/cert_\w*.pem;`)
					match := re.Find(config)
					Expect(match).NotTo(BeNil())
				})

				It("should have a valid ssl_certificate_key directive", func() {
					re := regexp.MustCompile(`[\r\n]\s*ssl_certificate_key\s*/tmp/key_\w*.pem;`)
					match := re.Find(config)
					Expect(match).NotTo(BeNil())
				})
			})

			Context("server listening on 61002", func() {
				It("should have a listen 61002 ssl directive", func() {
					Expect(false).Should(Equal(true))
				})

				It("should have a proxy_pass directive to sshd", func() {
					Expect(false).Should(Equal(true))
				})

				It("should have a valid ssl_certificate directive", func() {
					Expect(false).Should(Equal(true))
				})

				It("should have a valid ssl_certificate_key directive", func() {
					Expect(false).Should(Equal(true))
				})
			})

		})
	})

	AfterEach(func() {
		os.Remove(configFile)
	})
})
