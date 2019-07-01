package app_test

import (
	"bytes"

	"code.cloudfoundry.org/envoy-nginx/app"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cmd", func() {
	var (
		stdout *bytes.Buffer
		stderr *bytes.Buffer
		cmd    app.Cmd
	)

	BeforeEach(func() {
		stdout = &bytes.Buffer{}
		stderr = &bytes.Buffer{}

		cmd = app.NewCmd(stdout, stderr)
	})

	Describe("Run", func() {
		It("executes the binary app with the arguments its given", func() {
			err := cmd.Run("echo", "banana", "kiwi", "orange")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout.String()).To(ContainSubstring("banana kiwi orange"))
		})

		Context("running the command fails", func() {
			It("returns an error", func() {
				err := cmd.Run("not-a-real-command")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not-a-real-command"))
			})
		})
	})
})
