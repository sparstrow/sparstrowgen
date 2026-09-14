package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const linkScheme = "sparstrowgen"

// pairingWindow matches the server's pairing lifetime. While a pending
// credential is this fresh the daemon is probably waiting on a person approving
// it in the browser, so it checks back quickly instead of backing off.
const pairingWindow = 10 * time.Minute

func isPairingLink(arg string) bool {
	return strings.HasPrefix(strings.ToLower(arg), linkScheme+":")
}

// pairingRequest accepts exactly sparstrowgen://pair?request=<opaque>. The link
// arrives from a browser, so anything else is refused rather than interpreted.
func pairingRequest(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("read pairing link: %w", err)
	}
	if u.Scheme != linkScheme || u.Host != "pair" || strings.Trim(u.Path, "/") != "" {
		return "", errors.New("this is not a sparstrowgen pairing link")
	}
	q := u.Query()
	if len(q) != 1 || len(q["request"]) != 1 || q.Get("request") == "" {
		return "", errors.New("the pairing link is incomplete")
	}
	request := q.Get("request")
	if len(request) > 128 || strings.ContainsAny(request, " /\\\r\n") {
		return "", errors.New("the pairing link is not valid")
	}
	return request, nil
}

// claim spends a pairing request.
//
// It sends the credential this computer already holds, so the server can tell
// when the computer is already connected to the same account; then nothing
// changes and alreadyPaired is true. Otherwise the new credential is saved as
// pending, and only becomes this computer's credential once it is approved.
func claim(ctx context.Context, api, request, name string) (alreadyPaired bool, err error) {
	body, err := json.Marshal(map[string]string{
		"request": request, "name": name, "current": readCredential(credentialName),
	})
	if err != nil {
		return false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api+"/daemon/pair", bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("could not reach sparstrowgen: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8192))
	if res.StatusCode != http.StatusOK {
		var refused struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(raw, &refused) == nil && refused.Error != "" {
			return false, fmt.Errorf("pairing was refused: %s", refused.Error)
		}
		return false, fmt.Errorf("pairing was refused: %s", res.Status)
	}
	var reply struct {
		Credential    string `json:"credential"`
		AlreadyPaired bool   `json:"alreadyPaired"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return false, errors.New("pairing succeeded but sparstrowgen sent an unreadable reply")
	}
	if reply.AlreadyPaired {
		return true, nil
	}
	if reply.Credential == "" {
		return false, errors.New("pairing succeeded but sparstrowgen sent no credential")
	}
	if err := writeCredential(pendingName, reply.Credential); err != nil {
		return false, fmt.Errorf("save this computer's credential: %w", err)
	}
	return false, nil
}

// recentlyPaired is true while a pending credential may still be awaiting approval.
func recentlyPaired() bool {
	p, err := credentialPath(pendingName)
	if err != nil {
		return false
	}
	info, err := os.Stat(p)
	return err == nil && time.Since(info.ModTime()) < pairingWindow
}

func computerName() string {
	return env("COMPUTERNAME", hostname())
}

// pairCommand is the development path: `go run ./cmd/daemon pair -request X`.
func pairCommand(args []string) int {
	fs := flag.NewFlagSet("pair", flag.ContinueOnError)
	request := fs.String("request", "", "the request id from a sparstrowgen://pair link")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *request == "" {
		fmt.Fprintln(os.Stderr, "pair needs -request from a sparstrowgen://pair link")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	already, err := claim(ctx, serverAPI(), *request, computerName())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if already {
		fmt.Println("this computer is already connected to that account; nothing changed")
		return 0
	}
	fmt.Println("claimed; approve this computer in the browser, then start the daemon")
	return 0
}
