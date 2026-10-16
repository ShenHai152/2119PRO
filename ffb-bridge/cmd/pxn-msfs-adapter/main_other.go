//go:build !windows

package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	selfTest := flag.Bool("self-test", false, "validate version and lifecycle state handling")
	flag.Parse()
	if *selfTest {
		if err := runSelfTest(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	fmt.Println("PXNMSFSAdapter is available only on Windows")
}
