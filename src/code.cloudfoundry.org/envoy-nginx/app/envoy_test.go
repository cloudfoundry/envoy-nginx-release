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
	EnvoyConfig     = "../fixtures/cf_assets_envoy_config/envoy.yaml"
	SdsIdCreds      = "../fixtures/cf_assets_envoy_config/sds-id-cert-and-key.yaml"
	SdsC2CCreds     = "../fixtures/cf_assets_envoy_config/sds-c2c-cert-and-key.yaml"
	SdsIdValidation = "../fixtures/cf_assets_envoy_config/sds-id-validation-context.yaml"
)

var _ = Describe("App", func() {
	var (
		logger *fakes.Logger
		cmd    *fakes.Cmd
		tailer *fakes.Tailer

		nginxBinPath string
		nginxConfDir string

		application app.App
	)

	BeforeEach(func() {
		logger = &fakes.Logger{}
		cmd = &fakes.Cmd{}
		tailer = &fakes.Tailer{}

		var err error
		nginxBinPath, err = gexec.Build("code.cloudfoundry.org/envoy-nginx/fixtures/nginx")
		Expect(err).ToNot(HaveOccurred())

		nginxConfDir, err = ioutil.TempDir("", "nginx")
		Expect(err).ToNot(HaveOccurred())

		application = app.NewApp(logger, cmd, tailer, EnvoyConfig)
	})

	AfterEach(func() {
		gexec.CleanupBuildArtifacts()
		Expect(os.RemoveAll(nginxConfDir)).NotTo(HaveOccurred())
	})

	Describe("NginxPath", func() {
		Context("when nginx.exe is in the same path as our app", func() {
			PIt("returns the path to nginx.exe", func() {
				path, err := application.GetNginxPath()
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal(nginxBinPath))
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

	Describe("Run", func() {
		It("configures and starts nginx", func() {
			err := application.Run(nginxConfDir, nginxBinPath, SdsIdCreds, SdsC2CCreds, SdsIdValidation)
			Expect(err).NotTo(HaveOccurred())

			Expect(cmd.RunCall.Receives[0].Binary).To(Equal(nginxBinPath))
			Expect(cmd.RunCall.Receives[0].Args).To(ConsistOf(
				"-p", ContainSubstring("nginx"),
			))

			files, err := ioutil.ReadDir(nginxConfDir)
			Expect(err).ToNot(HaveOccurred())

			names := []string{}
			for _, file := range files {
				names = append(names, file.Name())
			}
			Expect(names).To(ConsistOf("logs", "conf", "id-cert.pem", "id-key.pem", "id-ca.pem", "c2c-cert.pem", "c2c-key.pem"))
		})

		Context("when running the command fails", func() {
			BeforeEach(func() {
				cmd.RunCall.Returns = []fakes.RunCallReturn{{Error: errors.New("banana")}}
			})

			It("returns a helpful error", func() {
				err := application.Run(nginxConfDir, nginxBinPath, SdsIdCreds, SdsC2CCreds, SdsIdValidation)
				Expect(err).To(MatchError("cmd run: banana"))

				Expect(logger.PrintlnCall.Messages).To(ContainElement(ContainSubstring("start nginx: ")))
			})
		})

		Context("when tailing error.log fails", func() {
			BeforeEach(func() {
				tailer.TailCall.Returns.Error = errors.New("banana")
			})

			It("returns a helpful error", func() {
				err := application.Run(nginxConfDir, nginxBinPath, SdsIdCreds, SdsC2CCreds, SdsIdValidation)
				Expect(err).To(MatchError("tail error log: banana"))
			})
		})

		Context("when nginx conf parser generate fails", func() {
			BeforeEach(func() {
				// TODO: When nginxConfParser is an object on app, then we can test it's errors
				// nginxConfParser.GenerateCall.Returns.Error = errors.New("banana")
			})

			PIt("returns a helpful error", func() {
				err := application.Run(nginxConfDir, nginxBinPath, SdsIdCreds, SdsC2CCreds, SdsIdValidation)
				Expect(err).To(MatchError("generate nginx config from envoy config: banana"))
			})
		})

		Context("when nginx conf parser write tls files fails", func() {
			BeforeEach(func() {
				// TODO: When nginxConfParser is an object on app, then we can test it's errors
				// nginxConfParser.WriteTlsFilesCall.Returns.Error = errors.New("banana")
			})

			PIt("returns a helpful error", func() {
				err := application.Run(nginxConfDir, nginxBinPath, SdsIdCreds, SdsC2CCreds, SdsIdValidation)
				Expect(err).To(MatchError("write tls files: banana"))
			})
		})
	})
})
