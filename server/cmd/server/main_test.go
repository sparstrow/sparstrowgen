package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestPostgresFailureDoesNotLogDatabasePassword(t *testing.T) {
	const (
		helperEnv = "SPARSTROWGEN_TEST_POSTGRES_FAILURE"
		password  = "database-password-must-not-appear"
	)

	if os.Getenv(helperEnv) == "1" {
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestPostgresFailureDoesNotLogDatabasePassword$")
	cmd.Env = append(os.Environ(),
		helperEnv+"=1",
		"DATABASE_URL=postgres://audit-user:"+password+"@127.0.0.1:1/audit?connect_timeout=1",
		"DAEMON_TOKEN=0123456789abcdef0123456789abcdef",
		"WEB_ORIGIN=https://app.example.com",
	)

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("server unexpectedly started with an unreachable database")
	}
	if !strings.Contains(string(output), "postgres unreachable") {
		t.Fatalf("failure log did not identify Postgres as unreachable:\n%s", output)
	}
	if strings.Contains(string(output), password) {
		t.Fatalf("failure log exposed the database password:\n%s", output)
	}
}
