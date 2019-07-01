package app_test

import (
	"errors"
	"io/ioutil"
	"os"

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
		var nginxConfDir string

		AfterEach(func() {
			err := os.RemoveAll(nginxConfDir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("configures and starts nginx", func() {
			err := application.Load(nginxPath, SdsCreds, SdsValidation)
			Expect(err).NotTo(HaveOccurred())

			Expect(cmd.RunCall.Receives[0].Binary).To(Equal(nginxPath))
			Expect(cmd.RunCall.Receives[0].Args).To(ConsistOf(
				"-c", ContainSubstring("envoy_nginx.conf"),
				"-p", ContainSubstring("nginx-conf"),
			))
			nginxConfDir = cmd.RunCall.Receives[0].Args[3]

			files, err := ioutil.ReadDir(nginxConfDir)
			Expect(err).ToNot(HaveOccurred())

			names := []string{}
			for _, file := range files {
				names = append(names, file.Name())
			}
			Expect(names).To(ConsistOf("logs", "envoy_nginx.conf", "cert.pem", "key.pem", "ca.pem"))
		})

		Context("when running the command fails", func() {
			BeforeEach(func() {
				cmd.RunCall.Returns = []fakes.RunCallReturn{{Error: errors.New("banana")}}
			})

			It("returns a helpful error", func() {
				err := application.Load(nginxPath, SdsCreds, SdsValidation)
				Expect(err).To(MatchError("cmd run: banana"))

				Expect(logger.PrintlnCall.Messages).To(ContainElement(ContainSubstring("start nginx: ")))

				nginxConfDir = cmd.RunCall.Receives[0].Args[3]
			})
		})

		Context("when nginx conf parser generate fails", func() {
			BeforeEach(func() {
				// TODO: When nginxConfParser is an object on app, then we can test it's errors
				// nginxConfParser.GenerateCall.Returns.Error = errors.New("banana")
			})

			PIt("returns a helpful error", func() {
				err := application.Load(nginxPath, SdsCreds, SdsValidation)
				Expect(err).To(MatchError("generate nginx config from envoy config: banana"))
			})
		})

		Context("when nginx conf parser write tls files fails", func() {
			BeforeEach(func() {
				// TODO: When nginxConfParser is an object on app, then we can test it's errors
				// nginxConfParser.WriteTlsFilesCall.Returns.Error = errors.New("banana")
			})

			PIt("returns a helpful error", func() {
				err := application.Load(nginxPath, SdsCreds, SdsValidation)
				Expect(err).To(MatchError("write tls files: banana"))
			})
		})
	})
})
