package testhelpers

import (
	"io/ioutil"
	"os"
)

func CopyFile(src, dst string) error {
	input, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}

	return nil
}

/*
* This function simulates how diego executor updates/rotates the sds file
* see github.com/cloudfoundry/executor/blob/0dc5df01a2e96e0d60cf285b880c5c2f4412e392/depot/containerstore/proxy_config_handler.go#L553-L558
 */
func RotateCert(newfile, sdsfilepath string) error {
	tmpPath := sdsfilepath + ".tmp"

	contents, err := ioutil.ReadFile(newfile)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(tmpPath, contents, 0666)
	if err != nil {
		return err
	}
	return os.Rename(tmpPath, sdsfilepath)
}
