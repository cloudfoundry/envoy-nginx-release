package app_test

import (
	"errors"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/envoy-nginx/app"
	"code.cloudfoundry.org/envoy-nginx/app/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	EnvoyConfig   = "../fixtures/cf_assets_envoy_config/envoy.yaml"
	SdsCreds      = "../fixtures/cf_assets_envoy_config/sds-server-cert-and-key.yaml"
	SdsValidation = "../fixtures/cf_assets_envoy_config/sds-server-validation-context.yaml"
)

var _ = Describe("App", func() {
	var (
		logger    *fakes.Logger
		cmd       *fakes.Cmd
		nginxPath string

		application app.App
	)

	BeforeEach(func() {
		logger = &fakes.Logger{}
		cmd = &fakes.Cmd{}

		var err error
		nginxPath, err = gexec.Build("code.cloudfoundry.org/envoy-nginx/fixtures/nginx")
		Expect(err).ToNot(HaveOccurred())

		application = app.NewApp(logger, cmd, EnvoyConfig)
	})

	AfterEach(func() {
		gexec.CleanupBuildArtifacts()
	})

	Describe("NginxPath", func() {
		Context("when nginx.exe is in the same path as our app", func() {
			BeforeEach(func() {
				withExe := filepath.Join(filepath.Dir(nginxPath), "nginx.exe")
				err := os.Rename(nginxPath, withExe)
				Expect(err).NotTo(HaveOccurred())

				nginxPath = withExe
			})

			AfterEach(func() {
				err := os.Remove(nginxPath)
				Expect(err).NotTo(HaveOccurred())
			})

			PIt("returns the path to nginx.exe", func() {
				path, err := application.GetNginxPath()
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal(nginxPath))
			})
		})

		Context("when nginx.exe cannot be found", func() {
			It("returns a helpful error", func() {
				path, err := application.GetNginxPath()
				Expect(err).To(MatchError(ContainSubstring("stat nginx.exe: ")))
				Expect(path).To(Equal(""))
			})
		})
	})

	Describe("Load", func() {
		It("loads the configurations for nginx and envoy", func() {
			err := application.Load(nginxPath, SdsCreds, SdsValidation)
			Expect(err).NotTo(HaveOccurred())

			Expect(cmd.RunCall.Receives[0].Binary).To(Equal(nginxPath))
			Expect(cmd.RunCall.Receives[0].Args).To(ConsistOf("-c", ContainSubstring("envoy_nginx.conf"), "-p", ContainSubstring("nginx-conf")))
		})

		Context("when running the command fails", func() {
			BeforeEach(func() {
				cmd.RunCall.Returns = []fakes.RunCallReturn{{Error: errors.New("banana")}}
			})

			It("returns a helpful error", func() {
				err := application.Load(nginxPath, SdsCreds, SdsValidation)
				Expect(err).To(MatchError("cmd run: banana"))
			})
		})
	})
})
