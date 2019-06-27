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
			application app.App
			envoyConfig string
			logger      *fakes.Logger
		)

		BeforeEach(func() {
			envoyConfig = "../fixtures/cf_assets_envoy_config/envoy.yaml"
			logger = &fakes.Logger{}
			application = app.NewApp(logger, envoyConfig)
		})

		It("loads the configurations for nginx and envoy", func() {
			err := application.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(logger.PrintlnCall.Messages).To(ConsistOf(
				"envoy.exe: Starting executable",
				fmt.Sprintf("Loading envoy config %s", envoyConfig),
			))
		})
	})
})
