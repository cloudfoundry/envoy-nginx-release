package main

import (
	"fmt"
	"io/ioutil"
	"os"

	. "code.cloudfoundry.org/envoy-nginx/testhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Watcher", func() {
	Context("when a file is watched", func() {
		var (
			watchmeFile string
			newFile     string
			err         error
		)

		BeforeEach(func() {
			watchmeFd, err := ioutil.TempFile("", "watchme")
			Expect(err).ToNot(HaveOccurred())
			watchmeFile = watchmeFd.Name()
			newFileFd, err := ioutil.TempFile("", "new-file")
			Expect(err).ToNot(HaveOccurred())
			newFile = newFileFd.Name()
		})

		It("Changes to the file must be detected every time", func() {
			ch := make(chan string)

			err = ioutil.WriteFile(watchmeFile, []byte("Heyy"), 0666)
			Expect(err).ToNot(HaveOccurred())

			/* I'm watchin U */
			go func() {
				WatchFile(watchmeFile, func() error {
					fmt.Println("WATCHER_FUNCTION_CALLED")
					ch <- "message"
					return nil
				})
			}()

			err = ioutil.WriteFile(newFile, []byte("Hello"), 0666)
			Expect(err).ToNot(HaveOccurred())
			RotateCert(newFile, watchmeFile)

			var str string
			Eventually(ch, "10s").Should(Receive(&str))
			Expect(str).Should(Equal("message"))
		})

		AfterEach(func() {
			os.Remove(watchmeFile)
			os.Remove(newFile)
		})
	})
})
