//go:build !windows

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
)

var errWindowsOnly = errors.New("installing and pairing links are only supported on Windows so far")

func install() error                  { return errWindowsOnly }
func activate(string) error           { return errWindowsOnly }
func singleInstance() (func(), bool)  { return func() {}, true }
func watchForExit(context.CancelFunc) {}
func ensureHiddenConsole()            {}
func notify(text string, _ bool)      { fmt.Fprintln(os.Stderr, text) }
