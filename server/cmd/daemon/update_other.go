//go:build !windows

package main

import (
	"fmt"
	"os"
)

func updatesSupported() bool                    { return false }
func startUpdate(string, string, string) error { return errWindowsOnly }

func applyUpdateCommand([]string) int {
	fmt.Fprintln(os.Stderr, errWindowsOnly)
	return 2
}
