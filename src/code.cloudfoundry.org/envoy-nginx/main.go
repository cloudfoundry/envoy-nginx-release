package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	/* Faker envoy.exe */
	output, err := exec.Command("./nginx.exe", "-c", "conf/nginx.conf").CombinedOutput()
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
	fmt.Println(string(output))
}
