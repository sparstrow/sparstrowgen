// Command server is the web-facing half: HTTP for everything the browser does,
// a websocket for what it watches, and one websocket the daemon dials in on.
//
// It never reaches out to anybody's machine. Nothing inbound to that machine
// exists at all — the daemon dials out (AGENTS.md §3).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	netmail "net/mail"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sparstrow/sparstrowgen/server/internal/api"
	"github.com/sparstrow/sparstrowgen/server/internal/hub"
	"github.com/sparstrow/sparstrowgen/server/internal/mail"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

// errServerRestarted is written into a turn that was running when the server
// stopped. Said as what happened, and what to do: the message can be sent again.
const errServerRestarted = "sparstrowgen's server restarted while this turn was running, so it was ended. Send the message again."

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// `server -healthcheck` asks the running server whether it is alive and
	// exits 0 or 1. It lives in this binary because the deployed image is
	// FROM scratch — there is no shell, no curl and no wget in it to run a
	// healthcheck with, which is the whole point of shipping it that way.
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(healthcheck())
	}

	dsn := env("DATABASE_URL",
		"postgres://sparstrowgen:sparstrowgen@localhost:5433/sparstrowgen?sslmode=disable")
	addr := env("ADDR", ":8080")
	cfg := serverConfig(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Error("connect to postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Error("postgres unreachable", "err", err)
		os.Exit(1)
	}

	h := hub.New(log)
	st := store.New(pool)
	a := api.New(st, h, log, cfg)

	// Before listening, so no turn can have started yet: any still open was
	// running when the last server stopped (docs/Bugs.md B-57).
	if n, err := st.EndOrphanedTurns(ctx, errServerRestarted); err != nil {
		log.Warn("could not close out turns left open by the last server", "err", err)
	} else if n > 0 {
		log.Info("closed out turns left open by the last server", "turns", n)
	}

	// Said once at startup, because it is the first thing to check when a new
	// deployment has nobody who can sign in: the owner creates their account
	// through the ordinary registration page with this address.
	if _, exists, err := st.UserByEmail(ctx, cfg.OwnerEmail); err != nil {
		log.Warn("could not tell whether the owner account exists", "err", err)
	} else if !exists {
		log.Info("the owner account does not exist yet — create it at /register with OWNER_EMAIL",
			"owner", cfg.OwnerEmail)
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: a.Routes(),
		// No write timeout: these are long-lived websockets, and a deadline
		// here would cut a turn off mid-answer.
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

// healthcheck calls the local server's liveness endpoint. Returns a process
// exit code: 0 alive, 1 not.
func healthcheck() int {
	addr := env("ADDR", ":8080")
	// ADDR is a listen address and may have no host at all (":8080") or a
	// wildcard one ("0.0.0.0:8080"); neither is dialable as written, and
	// 0.0.0.0 in particular is not an address you connect TO.
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: cannot read ADDR %q: %v\n", addr, err)
		return 1
	}
	client := &http.Client{Timeout: 2 * time.Second}
	res, err := client.Get("http://127.0.0.1:" + port + "/api/health")
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		return 1
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck: %s\n", res.Status)
		return 1
	}
	return 0
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

/*
	Configuration the server refuses to start without.

There is no default password, no "auth disabled" switch, no "mail off" and no
development shortcut, because every one of those is a thing that can end up in
production by accident — and what is behind this login is a program that runs
code on people's machines. A server that will not boot is a loud, immediate,
local failure. A server that boots without them is a silent, remote one.

Development is served by the Makefile setting these to known values, so the
cost of the strictness is one line in a file, not a branch in the program.
*/
func serverConfig(log *slog.Logger) api.Config {
	cfg := api.Config{
		Origin: os.Getenv("WEB_ORIGIN"),
		// Secure unless explicitly turned off, so forgetting it is the safe
		// mistake: a cookie that will not travel over http://localhost is an
		// obvious local annoyance, while one sent in the clear over the internet
		// is a quiet disaster.
		SecureCookie: os.Getenv("SESSION_SECURE") != "false",
		OwnerEmail:   strings.TrimSpace(os.Getenv("OWNER_EMAIL")),
		Invited:      splitList(os.Getenv("ALLOWED_EMAILS")),
	}

	// The shared daemon token is a development thing now (D-038). An installed
	// computer authenticates with the credential it was paired with and never
	// presents this, so on a deployment it would be a standing shared secret
	// that only somebody who should not have it would ever use. Ignored there,
	// rather than refused, so that setting it does not take a running
	// deployment down — the warning and the runbook say to remove it.
	token := os.Getenv("DAEMON_TOKEN")
	if !cfg.SecureCookie {
		cfg.DaemonToken = token
	} else if token != "" {
		log.Warn("DAEMON_TOKEN is set but ignored: a deployed server accepts only the credential a computer was paired with. Remove the variable.")
	}

	var missing []string
	if !cfg.SecureCookie && cfg.DaemonToken == "" {
		missing = append(missing, "DAEMON_TOKEN (any long random string, the same one the daemon uses; development only)")
	}
	if cfg.Origin == "" {
		missing = append(missing, "WEB_ORIGIN (the exact origin the web app is served from, e.g. https://app.sparstrow.com)")
	}
	if cfg.OwnerEmail == "" {
		missing = append(missing, "OWNER_EMAIL (the address of the account this deployment belongs to)")
	}
	transport := os.Getenv("MAIL_TRANSPORT")
	if transport == "" {
		missing = append(missing, "MAIL_TRANSPORT (smtp, or log on a development machine)")
	}
	if len(missing) > 0 {
		refuse(log, missing...)
	}

	if cfg.DaemonToken != "" && len(cfg.DaemonToken) < 32 {
		log.Error("DAEMON_TOKEN is too short to be a secret", "length", len(cfg.DaemonToken), "want_at_least", 32)
		os.Exit(1)
	}
	for _, address := range append([]string{cfg.OwnerEmail}, cfg.Invited...) {
		if _, err := netmail.ParseAddress(address); err != nil {
			log.Error("not an email address in OWNER_EMAIL or ALLOWED_EMAILS", "value", address)
			os.Exit(1)
		}
	}

	switch transport {
	case "smtp":
		port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
		sender := mail.SMTP{
			Host:     os.Getenv("SMTP_HOST"),
			Port:     port,
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     os.Getenv("MAIL_FROM"),
		}
		if err := sender.Validate(); err != nil {
			// The error names variables, never their values: a password in a
			// startup log is a password in every log aggregator it reaches.
			refuse(log, "MAIL_TRANSPORT=smtp needs working settings: "+err.Error())
		}
		cfg.Mailer = sender
	case "log":
		// Links written to a log are working credentials for whoever reads it.
		// Fine on a development machine; never on a deployment, which is the
		// one place SESSION_SECURE must be on.
		if cfg.SecureCookie {
			refuse(log, "MAIL_TRANSPORT=log is for development only and cannot be used with secure session cookies")
		}
		log.Warn("emails will be written to this log instead of sent (MAIL_TRANSPORT=log)")
		cfg.Mailer = mail.Log{Logger: log}
	default:
		refuse(log, "MAIL_TRANSPORT must be smtp or log, not "+strconv.Quote(transport))
	}
	return cfg
}

func refuse(log *slog.Logger, problems ...string) {
	log.Error("refusing to start without its configuration")
	for _, p := range problems {
		log.Error("  missing or invalid", "setting", p)
	}
	os.Exit(1)
}

// splitList reads a comma-separated list, ignoring blanks, so a trailing comma
// or a space after one is not an address.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
