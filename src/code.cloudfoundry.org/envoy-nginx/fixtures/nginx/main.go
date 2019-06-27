package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Println(strings.Join(os.Args, ","))

	time.Sleep(3 * time.Second)
}
