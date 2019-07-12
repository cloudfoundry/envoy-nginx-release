package app_test

import (
	"os"
	"path/filepath"

	"code.cloudfoundry.org/envoy-nginx/app"
	"code.cloudfoundry.org/envoy-nginx/app/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LogTailer", func() {
	Describe("Tail", func() {
		var (
			logTailer app.LogTailer
			errorLog  string
			logger    *fakes.Logger
		)

		BeforeEach(func() {
			logger = &fakes.Logger{}
			logTailer = app.NewLogTailer(logger)
			errorLog = filepath.Join(os.TempDir(), "error.log")
		})

		It("creates and tails logs/error.log", func() {
			err := logTailer.Tail(errorLog)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(errorLog)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when it cannot create the error.log", func() {
			It("returns a helpful error", func() {
				err := logTailer.Tail("/not-a-real-dir/not-a-real-file")
				Expect(err).To(MatchError(ContainSubstring("write error.log: ")))
			})
		})

		AfterEach(func() {
			os.Remove(errorLog)
		})
	})
})
