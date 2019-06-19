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

	for i, arg := range os.Args {
		if arg == "-s" && len(os.Args) > i+1 && os.Args[i+1] == "reload" {
			os.Exit(1)
		}
	}

	time.Sleep(1 * time.Second)
}
