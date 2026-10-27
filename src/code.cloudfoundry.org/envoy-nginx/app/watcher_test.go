package app_test

import (
	"errors"
	"fmt"
	"os"

	"code.cloudfoundry.org/envoy-nginx/app"
	. "code.cloudfoundry.org/envoy-nginx/testhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Watcher", func() {
	Describe("WatchFile", func() {
		var (
			watchmeFile string
			newFile     string
			err         error
		)

		BeforeEach(func() {
			watchmeFd, err := os.CreateTemp("", "watchme")
			Expect(err).ToNot(HaveOccurred())
			watchmeFile = watchmeFd.Name()
			watchmeFd.Close()

			newFileFd, err := os.CreateTemp("", "new-file")
			Expect(err).ToNot(HaveOccurred())
			newFile = newFileFd.Name()
			newFileFd.Close()
		})

		AfterEach(func() {
			Expect(os.Remove(watchmeFile)).NotTo(HaveOccurred())
			Expect(os.Remove(newFile)).NotTo(HaveOccurred())
		})

		It("detects changes to the file and executes the callback, repeatedly", func() {
			ch := make(chan string)
			readyChan := make(chan bool)

			err = os.WriteFile(watchmeFile, []byte("Heyy"), 0666)
			Expect(err).ToNot(HaveOccurred())

			go func() {
				app.WatchFile(watchmeFile, readyChan, func() error {
					ch <- "message"
					return nil
				})
			}()

			<-readyChan

			for i := 0; i <= 3; i++ {
				content := fmt.Sprintf("Hello-%d\n", i)
				err = os.WriteFile(newFile, []byte(content), 0666)
				Expect(err).ToNot(HaveOccurred())
				err = RotateCert(newFile, watchmeFile)
				Expect(err).ToNot(HaveOccurred())

				var str string
				Eventually(ch, "10s").Should(Receive(&str))
				Expect(str).Should(Equal("message"))
			}
		})

		Context("when several watchers share a readyChan that is only read once", func() {
			var (
				watchedFiles []string
				newFiles     []string
			)

			BeforeEach(func() {
				watchedFiles = []string{watchmeFile}
				newFiles = []string{newFile}

				watchmeFd, err := os.CreateTemp("", "watchme-2")
				Expect(err).ToNot(HaveOccurred())
				watchedFiles = append(watchedFiles, watchmeFd.Name())
				watchmeFd.Close()

				newFileFd, err := os.CreateTemp("", "new-file-2")
				Expect(err).ToNot(HaveOccurred())
				newFiles = append(newFiles, newFileFd.Name())
				newFileFd.Close()
			})

			AfterEach(func() {
				Expect(os.Remove(watchedFiles[1])).NotTo(HaveOccurred())
				Expect(os.Remove(newFiles[1])).NotTo(HaveOccurred())
			})

			It("returns the callback error from every watcher instead of blocking", func() {
				readyChan := make(chan bool)
				errCh := make(chan error, len(watchedFiles))

				for _, f := range watchedFiles {
					go func() {
						errCh <- app.WatchFile(f, readyChan, func() error {
							return errors.New("callback failed")
						})
					}()
				}

				// Consumed exactly once, as in envoy.go
				<-readyChan

				// The watcher that lost the readyChan send may not have called
				// watcher.Add yet, so keep rotating until every watcher reports.
				got := 0
				Eventually(func() int {
					for i := range watchedFiles {
						Expect(os.WriteFile(newFiles[i], []byte("rotated"), 0666)).To(Succeed())
						Expect(RotateCert(newFiles[i], watchedFiles[i])).To(Succeed())
					}
					for {
						select {
						case err := <-errCh:
							Expect(err).To(MatchError("callback failed"))
							got++
						default:
							return got
						}
					}
				}, "5s", "100ms").Should(Equal(len(watchedFiles)))
			})
		})
	})
})
