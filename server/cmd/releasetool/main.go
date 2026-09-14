// Command releasetool writes the manifest installed computers check for updates
// (docs/Decisions.md D-035). Used by scripts/package-windows.ps1.
//
//	releasetool manifest -exe <installer> -version <x.y.z> -url <installer URL> -out <dir>
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sparstrow/sparstrowgen/server/internal/release"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "manifest" {
		usage()
	}
	fs := flag.NewFlagSet("manifest", flag.ExitOnError)
	exe := fs.String("exe", "", "installer to describe")
	version := fs.String("version", "", "release version, x.y.z")
	url := fs.String("url", "", "https URL the installer is published at")
	out := fs.String("out", "", "directory to write the manifest to")
	_ = fs.Parse(os.Args[2:])
	if *exe == "" || *version == "" || *url == "" || *out == "" {
		usage()
	}

	sum, err := release.FileSHA256(*exe)
	exitOn(err)
	body, err := release.Encode(release.Manifest{Version: *version, URL: *url, SHA256: sum})
	exitOn(err)
	exitOn(os.MkdirAll(*out, 0o755))
	manifest := filepath.Join(*out, "sparstrowgen-update.json")
	exitOn(os.WriteFile(manifest, body, 0o644))
	fmt.Println(manifest)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: releasetool manifest -exe <installer> -version <x.y.z> -url <https URL> -out <dir>")
	os.Exit(2)
}

func exitOn(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
