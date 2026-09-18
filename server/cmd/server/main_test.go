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
// serverEnv is the environment for a server started by these tests: the
// machine's own, minus anything sparstrowgen reads, plus what the test sets.
//
// The subtraction is the point, and it was found the hard way — this machine
// had DAEMON_TOKEN set as a user environment variable, so a test asserting that
// a server starts WITHOUT one was quietly handed one and failed. A test whose
// answer depends on the developer's shell is not testing the program.
func serverEnv(settings ...string) []string {
	ours := map[string]bool{}
	for _, s := range settings {
		if k, _, found := strings.Cut(s, "="); found {
			ours[k] = true
		}
	}
	var env []string
	for _, entry := range os.Environ() {
		k, _, _ := strings.Cut(entry, "=")
		switch {
		case ours[k]:
			continue // the test sets it below
		case k == "DAEMON_TOKEN" || k == "SESSION_SECURE" || k == "WEB_ORIGIN",
			k == "OWNER_EMAIL" || k == "ALLOWED_EMAILS" || k == "DATABASE_URL",
			strings.HasPrefix(k, "SMTP_"), strings.HasPrefix(k, "MAIL_"):
			continue
		}
		env = append(env, entry)
	}
	return append(env, settings...)
}

// A deployment no longer needs the shared daemon token, and says so if one is
// set: computers authenticate with the credential they were paired with, so on
// a deployment the token is a standing secret nothing legitimate uses (D-038).
//
// It starts rather than refusing on purpose. Refusing would take down a running
// deployment the moment this version reached it, for a variable that is merely
// unnecessary.
func TestADeploymentStartsWithoutADaemonTokenAndIgnoresOne(t *testing.T) {
	const helperEnv = "SPARSTROWGEN_TEST_DEPLOYED_TOKEN"

	if os.Getenv(helperEnv) == "1" {
		main()
		return
	}

	for _, given := range []struct {
		name, token, want string
	}{
		{"none set", "", ""},
		{"one set", "DAEMON_TOKEN=0123456789abcdef0123456789abcdef", "ignored"},
	} {
		t.Run(given.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestADeploymentStartsWithoutADaemonTokenAndIgnoresOne$")
			cmd.Env = serverEnv(
				helperEnv+"=1",
				"WEB_ORIGIN=https://app.example.com",
				"OWNER_EMAIL=owner@example.com",
				"MAIL_TRANSPORT=smtp",
				"SMTP_HOST=smtp.example.com", "SMTP_PORT=465",
				"SMTP_USERNAME=owner@example.com", "SMTP_PASSWORD=not-a-real-password",
				"MAIL_FROM=sparstrowgen <owner@example.com>",
				"SESSION_SECURE=true",
				// No database, so it gets as far as configuration and then
				// fails on Postgres — which is all this needs to see.
				"DATABASE_URL=postgres://nobody@127.0.0.1:1/none?sslmode=disable",
			)
			if given.token != "" {
				cmd.Env = append(cmd.Env, given.token)
			}
			output, _ := cmd.CombinedOutput()

			if strings.Contains(string(output), "DAEMON_TOKEN (any long random string") {
				t.Errorf("a deployment still demands DAEMON_TOKEN:\n%s", output)
			}
			if given.want == "" && strings.Contains(string(output), "DAEMON_TOKEN") {
				t.Errorf("a deployment with no token mentioned it anyway:\n%s", output)
			}
			if given.want == "ignored" && !strings.Contains(string(output), "ignored") {
				t.Errorf("a deployment did not say the token it was given is ignored:\n%s", output)
			}
		})
	}
}

// Development is the one place the shared token still works, and it is still
// required there — the development daemon has no pairing and nothing else to
// present.
func TestADevelopmentServerStillRequiresTheDaemonToken(t *testing.T) {
	const helperEnv = "SPARSTROWGEN_TEST_DEV_TOKEN"

	if os.Getenv(helperEnv) == "1" {
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestADevelopmentServerStillRequiresTheDaemonToken$")
	cmd.Env = serverEnv(
		helperEnv+"=1",
		"WEB_ORIGIN=http://localhost:3000",
		"OWNER_EMAIL=owner@localhost.test",
		"MAIL_TRANSPORT=log",
		"SESSION_SECURE=false",
		"DATABASE_URL=postgres://nobody@127.0.0.1:1/none?sslmode=disable",
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("a development server started with no daemon token")
	}
	if !strings.Contains(string(output), "DAEMON_TOKEN") {
		t.Fatalf("the refusal did not name the missing variable:\n%s", output)
	}
}

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
