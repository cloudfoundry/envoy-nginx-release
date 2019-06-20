package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
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
			watchmeFd.Close()
			newFileFd, err := ioutil.TempFile("", "new-file")
			Expect(err).ToNot(HaveOccurred())
			newFile = newFileFd.Name()
		})

		It("detects changes to the file and executes the callback", func() {
			ch := make(chan string)
			readyChan := make(chan bool)

			err = ioutil.WriteFile(watchmeFile, []byte("Heyy"), 0666)
			Expect(err).ToNot(HaveOccurred())

			go func() {
				WatchFile(watchmeFile, readyChan, func() error {
					ch <- "message"
					return nil
				})
			}()

			<-readyChan
			content := fmt.Sprintf("Hello-%d\n", rand.Intn(1000))
			err = ioutil.WriteFile(newFile, []byte(content), 0666)
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
