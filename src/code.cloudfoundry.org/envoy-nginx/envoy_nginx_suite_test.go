package main_test

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const sdsFixture = "fixtures/cf_assets_envoy_config/sds-server-cert-and-key.yaml"

var (
	envoyNginxBin string
	err           error
	binParentDir  string
	nginxBin      string
)

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

var _ = BeforeSuite(func() {
	envoyNginxBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx")
	Expect(err).ToNot(HaveOccurred())

	binParentDir, err = ioutil.TempDir("", "envoy-nginx")
	Expect(err).ToNot(HaveOccurred())

	basename := filepath.Base(envoyNginxBin)
	err = os.Rename(envoyNginxBin, filepath.Join(binParentDir, basename))
	Expect(err).ToNot(HaveOccurred())
	envoyNginxBin = filepath.Join(binParentDir, basename)

	nginxBin, err = gexec.Build("code.cloudfoundry.org/envoy-nginx/fixtures/nginx")
	Expect(err).ToNot(HaveOccurred())

	err = os.Rename(nginxBin, filepath.Join(binParentDir, "nginx.exe"))
	Expect(err).ToNot(HaveOccurred())
	nginxBin = filepath.Join(binParentDir, "nginx.exe")
})

var _ = AfterSuite(func() {
	os.RemoveAll(binParentDir)
})
