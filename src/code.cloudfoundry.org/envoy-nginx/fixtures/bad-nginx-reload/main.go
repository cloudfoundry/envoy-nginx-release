/* Program to simulate an nginx that always fails */
package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Println(strings.Join(os.Args, ","))

	if len(os.Args) > 2 && os.Args[1] == "-s" && os.Args[2] == "reload" {
		os.Exit(1)
	}

	// TODO: fix this, it could be a flake
	time.Sleep(1 * time.Second)
}
