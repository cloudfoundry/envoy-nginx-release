package app_test

import (
	"fmt"

	"code.cloudfoundry.org/envoy-nginx/app"
	"code.cloudfoundry.org/envoy-nginx/app/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("App", func() {
	Describe("Load", func() {
		var (
			application   app.App
			envoyConfig   string
			logger        *fakes.Logger
			sdsCreds      string
			sdsValidation string
		)

		BeforeEach(func() {
			envoyConfig = "../fixtures/cf_assets_envoy_config/envoy.yaml"
			sdsCreds = "../fixtures/cf_assets_envoy_config/sds-server-cert-and-key.yaml"
			sdsValidation = "../fixtures/cf_assets_envoy_config/sds-server-validation-context.yaml"
			logger = &fakes.Logger{}
			application = app.NewApp(logger, envoyConfig)
		})

		PIt("loads the configurations for nginx and envoy", func() {
			err := application.Load(sdsCreds, sdsValidation)
			Expect(err).NotTo(HaveOccurred())
			Expect(logger.PrintlnCall.Messages).To(ConsistOf(
				"envoy.exe: Starting executable",
				fmt.Sprintf("Loading envoy config %s", envoyConfig),
			))
		})
	})
})
