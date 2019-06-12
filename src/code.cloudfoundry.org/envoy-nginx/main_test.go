package main_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Envoy-Nginx", func() {
	Context("when nginx.exe is present in the same directory", func() {
		var (
			args            []string
			outputDirectory string
		)

		BeforeEach(func() {
			os.Setenv("ENVOY_FILE", "fixtures/cf_assets_envoy_config/envoy.yaml")
			os.Setenv("SDS_FILE", "fixtures/cf_assets_envoy_config/sds-server-cert-and-key.yaml")

			stdout, _, err := Execute(exec.Command(envoyNginxBin))
			Expect(err).ToNot(HaveOccurred())

			args = strings.Split(string(stdout.Bytes()), ",")
			/*
			* TODO: see about cleaning up output directory.
			* There's a risk that if we get it wrong, we end up deleting
			* some random directory on our filesystem.
			 */
			outputDirectory = strings.TrimSpace(args[3])
		})

		It("calls nginx with the right arguments", func() {
			Expect(strings.TrimSpace(args[0])).To(Equal("-c"))

			nginxConf := strings.TrimSpace(args[1])
			_, err = os.Stat(nginxConf)
			Expect(err).ToNot(HaveOccurred())

			Expect(strings.TrimSpace(args[2])).To(Equal("-p"))

			Expect(outputDirectory).ToNot(BeEmpty())
		})

		It("creates the right files in the output directory", func() {
			files, err := ioutil.ReadDir(outputDirectory)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(files)).To(Equal(3))

			var foundConf, foundCert, foundKey bool

			for _, file := range files {
				if file.Name() == "envoy_nginx.conf" {
					foundConf = true
				}
				if file.Name() == "cert.pem" {
					foundCert = true
				}
				if file.Name() == "key.pem" {
					foundKey = true
				}
			}

			Expect(foundConf).To(BeTrue())
			Expect(foundCert).To(BeTrue())
			Expect(foundKey).To(BeTrue())
		})
	})

	Context("when nginx.exe is not present in the same directory", func() {
		var (
			aloneBin       string
			aloneParentDir string
		)

		BeforeEach(func() {
			aloneBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx")
			Expect(err).ToNot(HaveOccurred())

			aloneParentDir, err = ioutil.TempDir("", "envoy-nginx")
			Expect(err).ToNot(HaveOccurred())

			basename := filepath.Base(aloneBin)
			os.Rename(aloneBin, filepath.Join(aloneParentDir, basename))
			Expect(err).ToNot(HaveOccurred())

			aloneBin = filepath.Join(aloneParentDir, basename)
		})

		It("errors", func() {
			_, _, err = Execute(exec.Command(aloneBin))
			Expect(err).To(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(aloneParentDir)
		})
	})
})

func Execute(c *exec.Cmd) (*bytes.Buffer, *bytes.Buffer, error) {
	stdOut := new(bytes.Buffer)
	stdErr := new(bytes.Buffer)
	c.Stdout = io.MultiWriter(stdOut, GinkgoWriter)
	c.Stderr = io.MultiWriter(stdErr, GinkgoWriter)
	err := c.Run()

	return stdOut, stdErr, err
}
