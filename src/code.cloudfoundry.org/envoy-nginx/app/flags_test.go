package app_test

import (
	"code.cloudfoundry.org/envoy-nginx/app"
	. "github.com/onsi/ginkgo"
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
			"--creds", SdsCreds,
			"--validation", SdsValidation,
		}
		flags = app.NewFlags()
	})

	It("parses known flags and returns options", func() {
		opts := flags.Parse(args)
		Expect(opts.EnvoyConfig).To(Equal(EnvoyConfig))
		Expect(opts.SdsCreds).To(Equal(SdsCreds))
		Expect(opts.SdsValidation).To(Equal(SdsValidation))
	})

	It("has defaults", func() {
		opts := flags.Parse([]string{})
		Expect(opts.EnvoyConfig).To(Equal(app.DefaultEnvoyConfigPath))
		Expect(opts.SdsCreds).To(Equal(app.DefaultSdsCertAndKeysPath))
		Expect(opts.SdsValidation).To(Equal(app.DefaultSdsValidationContextPath))
	})

	It("does not fail with unknown flags", func() {
		opts := flags.Parse([]string{"--invalid", "invalid"})
		Expect(opts.EnvoyConfig).To(Equal(app.DefaultEnvoyConfigPath))
	})

	Context("when provided a flag with no argument", func() {
		It("continues to use the default", func() {
			opts := flags.Parse([]string{"-c", "--creds", "--validation", "--invalid", "invalid"})
			Expect(opts.EnvoyConfig).To(Equal(app.DefaultEnvoyConfigPath))
		})
	})
})
