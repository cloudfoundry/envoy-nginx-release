/* Program to simulate an nginx that always fails */
package main

import (
	"os"
	"time"
)

func main() {
	time.Sleep(1 * time.Second)
	os.Exit(1)
}
