package main_test

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const sdsFixture = "fixtures/cf_assets_envoy_config/sds-server-cert-and-key.yaml"

func TestEnvoyNginx(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EnvoyNginx Suite")
}

func copyFile(src, dst string) error {
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}
	return nil
}

func Execute(c *exec.Cmd) (*bytes.Buffer, *bytes.Buffer, error) {
	stdOut := new(bytes.Buffer)
	stdErr := new(bytes.Buffer)
	c.Stdout = io.MultiWriter(stdOut, GinkgoWriter)
	c.Stderr = io.MultiWriter(stdErr, GinkgoWriter)
	err := c.Run()

	return stdOut, stdErr, err
}

func Start(c *exec.Cmd) (*gexec.Session, error) {
	stdOut := new(bytes.Buffer)
	stdErr := new(bytes.Buffer)
	c.Stdout = io.MultiWriter(stdOut, GinkgoWriter)
	c.Stderr = io.MultiWriter(stdErr, GinkgoWriter)
	session, err := gexec.Start(c, GinkgoWriter, GinkgoWriter)

	return session, err
}
