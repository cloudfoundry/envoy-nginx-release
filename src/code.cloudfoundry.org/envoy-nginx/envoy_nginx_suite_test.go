package main_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

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
