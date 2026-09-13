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
		"OWNER_EMAIL=owner@example.com",
		"MAIL_TRANSPORT=log",
		"SESSION_SECURE=false",
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

// Mail credentials must never reach the log, even when the configuration is
// refused — the refusal names the variables and nothing else.
func TestRefusedMailSettingsDoNotLogThePassword(t *testing.T) {
	const (
		helperEnv = "SPARSTROWGEN_TEST_MAIL_REFUSED"
		password  = "smtp-password-must-not-appear"
	)

	if os.Getenv(helperEnv) == "1" {
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestRefusedMailSettingsDoNotLogThePassword$")
	cmd.Env = append(os.Environ(),
		helperEnv+"=1",
		"DAEMON_TOKEN=0123456789abcdef0123456789abcdef",
		"WEB_ORIGIN=https://app.example.com",
		"OWNER_EMAIL=owner@example.com",
		"MAIL_TRANSPORT=smtp",
		"SMTP_HOST=smtp.example.com",
		"SMTP_PASSWORD="+password,
		// No port, username or sender: refused.
	)

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("server started with incomplete SMTP settings")
	}
	if !strings.Contains(string(output), "SMTP_PORT") {
		t.Fatalf("the refusal did not name what is missing:\n%s", output)
	}
	if strings.Contains(string(output), password) {
		t.Fatalf("the refusal exposed the SMTP password:\n%s", output)
	}
}

// The log transport writes working links into the log, which is fine on a
// development machine and never on a deployment.
func TestLogMailIsRefusedWithSecureCookies(t *testing.T) {
	const helperEnv = "SPARSTROWGEN_TEST_LOG_MAIL_SECURE"

	if os.Getenv(helperEnv) == "1" {
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestLogMailIsRefusedWithSecureCookies$")
	cmd.Env = append(os.Environ(),
		helperEnv+"=1",
		"DAEMON_TOKEN=0123456789abcdef0123456789abcdef",
		"WEB_ORIGIN=https://app.example.com",
		"OWNER_EMAIL=owner@example.com",
		"MAIL_TRANSPORT=log",
		"SESSION_SECURE=true",
	)

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("server started writing emails to the log with secure cookies on")
	}
	if !strings.Contains(string(output), "development only") {
		t.Fatalf("the refusal did not say why:\n%s", output)
	}
}
