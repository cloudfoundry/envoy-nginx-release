package app

import (
	"io"
	"os/exec"
)

type Cmd struct {
	stdout io.Writer
	stderr io.Writer
}

func NewCmd(stdout, stderr io.Writer) Cmd {
	return Cmd{
		stdout: stdout,
		stderr: stderr,
	}
}

func (c Cmd) Run(binary string, arg ...string) (err error) {
	cmd := exec.Command(binary, arg...)
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr
	return cmd.Run()
}
