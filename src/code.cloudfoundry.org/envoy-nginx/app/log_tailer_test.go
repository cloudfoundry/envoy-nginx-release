package app_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/envoy-nginx/app"
	"code.cloudfoundry.org/envoy-nginx/app/fakes"
	"github.com/hpcloud/tail/watch"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Log Tailer", func() {
	Describe("Tail error log", func() {
		var (
			logTailer app.LogTailer
			errorLog  string
			logger    *fakes.Logger
		)

		BeforeEach(func() {
			logger = &fakes.Logger{}
			logTailer = app.NewLogTailer(logger)
			errorLog = filepath.Join(os.TempDir(), "error.log")

			watch.POLL_DURATION = (5 * time.Millisecond)
		})

		// TODO: Solve race condition introduced by spinning it out as a goroutine.
		PIt("tails logs/error.log", func() {
			go logTailer.Tail(errorLog)

			time.Sleep(watch.POLL_DURATION)

			_, err := os.Stat(errorLog)
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile(errorLog, []byte("bananas in pajamas\nare running down the stairs\n"), 0755|os.ModeAppend)
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(watch.POLL_DURATION)

			Expect(logger.PrintlnCall.Messages[0]).To(Equal("bananas in pajamas"))
			Expect(logger.PrintlnCall.Messages[1]).To(Equal("are running down the stairs"))
		})

		AfterEach(func() {
			os.Remove(errorLog)
		})
	})
})
