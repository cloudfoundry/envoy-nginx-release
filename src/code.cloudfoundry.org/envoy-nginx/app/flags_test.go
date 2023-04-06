package app_test

import (
	"code.cloudfoundry.org/envoy-nginx/app"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Flags", func() {
	var (
		args  []string
		flags app.Flags
	)

	BeforeEach(func() {
		args = []string{
			"-c", EnvoyConfig,
			"--id-creds", SdsIdCreds,
			"--c2c-creds", SdsC2CCreds,
			"--id-validation", SdsIdValidation,
		}
		flags = app.NewFlags()
	})

	Describe("Parse", func() {
		It("parses known flags and returns options", func() {
			opts := flags.Parse(args)
			Expect(opts.EnvoyConfig).To(Equal(EnvoyConfig))
			Expect(opts.SdsIdCreds).To(Equal(SdsIdCreds))
			Expect(opts.SdsC2CCreds).To(Equal(SdsC2CCreds))
			Expect(opts.SdsIdValidation).To(Equal(SdsIdValidation))
		})

		It("has defaults", func() {
			opts := flags.Parse([]string{})
			Expect(opts.EnvoyConfig).To(Equal(app.DefaultEnvoyConfigPath))
			Expect(opts.SdsIdCreds).To(Equal(app.DefaultSdsIdCertAndKeysPath))
			Expect(opts.SdsC2CCreds).To(BeEmpty())
			Expect(opts.SdsIdValidation).To(Equal(app.DefaultSdsIdValidationContextPath))
		})

		It("does not fail with unknown flags", func() {
			opts := flags.Parse([]string{"--invalid", "invalid"})
			Expect(opts.EnvoyConfig).To(Equal(app.DefaultEnvoyConfigPath))
		})

		Context("when provided a flag with no argument", func() {
			It("continues to use the default", func() {
				opts := flags.Parse([]string{"-c", "--id-creds", "--id-validation", "--invalid", "invalid"})
				Expect(opts.EnvoyConfig).To(Equal(app.DefaultEnvoyConfigPath))
			})
		})
	})
})
