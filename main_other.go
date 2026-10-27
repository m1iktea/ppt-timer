//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "ppt-timer 只支持 Windows")
	os.Exit(1)
}
