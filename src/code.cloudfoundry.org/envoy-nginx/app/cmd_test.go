package app_test

import (
	"bytes"
	"runtime"

	"code.cloudfoundry.org/envoy-nginx/app"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cmd", func() {
	var (
		stdout *bytes.Buffer
		stderr *bytes.Buffer
		bin    string
		args   []string

		cmd app.Cmd
	)

	BeforeEach(func() {
		stdout = &bytes.Buffer{}
		stderr = &bytes.Buffer{}

		if runtime.GOOS == "windows" {
			bin = "powershell"
			args = []string{"echo", "banana"}
		} else {
			bin = "echo"
			args = []string{"banana"}
		}

		cmd = app.NewCmd(stdout, stderr)
	})

	Describe("Run", func() {
		It("executes the binary app with the arguments its given", func() {
			err := cmd.Run(bin, args...)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout.String()).To(ContainSubstring("banana"))
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
