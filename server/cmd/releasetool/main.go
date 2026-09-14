// Command releasetool makes the signed manifest installed computers check for
// updates (docs/Decisions.md D-034). Used by scripts/package-windows.ps1.
//
//	releasetool keygen   -key <file>
//	releasetool pubkey   -key <file>
//	releasetool manifest -key <file> -exe <installer> -version <x.y.z> -url <installer URL> -out <dir>
//
// The signing key never enters the repository. Its public half is built into
// every release, so replacing it strands installed computers on their current
// version until someone installs a new one by hand.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sparstrow/sparstrowgen/server/internal/release"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	keyPath := fs.String("key", "", "signing key file")
	exe := fs.String("exe", "", "installer to describe")
	version := fs.String("version", "", "release version, x.y.z")
	url := fs.String("url", "", "https URL the installer is published at")
	out := fs.String("out", "", "directory to write the manifest and signature to")
	_ = fs.Parse(os.Args[2:])
	if *keyPath == "" {
		usage()
	}

	switch os.Args[1] {
	case "keygen":
		pub, err := release.GenerateKey(*keyPath)
		exitOn(err)
		fmt.Println(pub)
	case "pubkey":
		key, err := release.LoadKey(*keyPath)
		exitOn(err)
		fmt.Println(release.PublicKey(key))
	case "manifest":
		if *exe == "" || *version == "" || *url == "" || *out == "" {
			usage()
		}
		key, err := release.LoadKey(*keyPath)
		exitOn(err)
		sum, err := release.FileSHA256(*exe)
		exitOn(err)
		body, sig, err := release.Sign(key, release.Manifest{Version: *version, URL: *url, SHA256: sum})
		exitOn(err)
		exitOn(os.MkdirAll(*out, 0o755))
		manifest := filepath.Join(*out, "sparstrowgen-update.json")
		exitOn(os.WriteFile(manifest, body, 0o644))
		exitOn(os.WriteFile(manifest+".sig", sig, 0o644))
		fmt.Println(manifest)
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: releasetool keygen|pubkey -key <file>\n       releasetool manifest -key <file> -exe <installer> -version <x.y.z> -url <https URL> -out <dir>")
	os.Exit(2)
}

func exitOn(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
