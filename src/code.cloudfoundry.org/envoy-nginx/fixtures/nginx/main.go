package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Println(strings.Join(os.Args, ","))

	// TODO: fix this, it could be a flake
	time.Sleep(10 * time.Second)
}
