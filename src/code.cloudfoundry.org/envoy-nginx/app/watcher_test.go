package app_test

import (
	"fmt"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/envoy-nginx/app"
	. "code.cloudfoundry.org/envoy-nginx/testhelpers"
	. "github.com/onsi/ginkgo"
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
			watchmeFd, err := ioutil.TempFile("", "watchme")
			Expect(err).ToNot(HaveOccurred())
			watchmeFile = watchmeFd.Name()
			watchmeFd.Close()
			newFileFd, err := ioutil.TempFile("", "new-file")
			Expect(err).ToNot(HaveOccurred())
			newFile = newFileFd.Name()
		})

		It("detects changes to the file and executes the callback, repeatedly", func() {
			ch := make(chan string)
			readyChan := make(chan bool)

			err = ioutil.WriteFile(watchmeFile, []byte("Heyy"), 0666)
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
				err = ioutil.WriteFile(newFile, []byte(content), 0666)
				Expect(err).ToNot(HaveOccurred())
				err = RotateCert(newFile, watchmeFile)
				Expect(err).ToNot(HaveOccurred())

				var str string
				Eventually(ch, "10s").Should(Receive(&str))
				Expect(str).Should(Equal("message"))
			}
		})

		AfterEach(func() {
			Expect(os.Remove(watchmeFile)).NotTo(HaveOccurred())
			Expect(os.Remove(newFile)).NotTo(HaveOccurred())
		})
	})
})
