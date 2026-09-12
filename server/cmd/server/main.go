// Command server is the web-facing half: HTTP for everything the browser does,
// a websocket for what it watches, and one websocket the daemon dials in on.
//
// It never reaches out to the owner's machine. Nothing inbound to that machine
// exists at all — the daemon dials out (AGENTS.md §3).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sparstrow/sparstrowgen/server/internal/api"
	"github.com/sparstrow/sparstrowgen/server/internal/hub"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

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
	cfg := authConfig(log)

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

	// The setup code is only meaningful while nobody has signed up. Printed
	// here, loudly, because the deployment log is where the owner reads it —
	// and NOT printed once the app is claimed, so it never sits in a log of a
	// running system where it would be a credential with nothing to protect.
	// A failed check prints the code as well. If the database was briefly
	// unreachable at startup and recovers, the app would otherwise offer the
	// sign-up screen while the only code that could complete it was never
	// printed — unclaimable without a restart. Printing it when the answer is
	// unknown costs nothing: if the app IS claimed, the code opens nothing,
	// because sign-up answers 409 whatever is presented.
	if claimed, err := st.Claimed(ctx); err != nil {
		log.Warn("could not tell whether this app has been claimed yet", "err", err)
		log.Info("printing the setup code anyway, in case it has not been",
			"setup_code", a.SetupCode())
	} else if !claimed {
		log.Info("this app has no account yet")
		log.Info("open it in a browser and use this setup code to create one",
			"setup_code", a.SetupCode())
		log.Info("the code changes every time this server restarts; use the newest one")
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
	Authentication configuration, which the server refuses to start without.

There is no default password, no "auth disabled" switch and no development
shortcut, because every one of those is a thing that can end up in production by
accident — and what is behind this login is a program that runs code on the
owner's machine. A server that will not boot is a loud, immediate, local
failure. A server that boots without a password is a silent, remote one.

Development is served by the Makefile setting these to known values, so the
cost of the strictness is one line in a file, not a branch in the program.
*/
func authConfig(log *slog.Logger) api.Config {
	cfg := api.Config{
		DaemonToken: os.Getenv("DAEMON_TOKEN"),
		Origin:      os.Getenv("WEB_ORIGIN"),
		// Secure unless explicitly turned off, so forgetting it is the safe
		// mistake: a cookie that will not travel over http://localhost is an
		// obvious local annoyance, while one sent in the clear over the internet
		// is a quiet disaster.
		SecureCookie: os.Getenv("SESSION_SECURE") != "false",
	}

	var missing []string
	if cfg.DaemonToken == "" {
		missing = append(missing, "DAEMON_TOKEN (any long random string, the same one the daemon uses)")
	}
	if cfg.Origin == "" {
		missing = append(missing, "WEB_ORIGIN (the exact origin the web app is served from, e.g. https://sparstrowgen.example.ts.net)")
	}
	if len(missing) > 0 {
		log.Error("refusing to start without authentication configured")
		for _, m := range missing {
			log.Error("  missing", "variable", m)
		}
		os.Exit(1)
	}

	if len(cfg.DaemonToken) < 32 {
		log.Error("DAEMON_TOKEN is too short to be a secret", "length", len(cfg.DaemonToken), "want_at_least", 32)
		os.Exit(1)
	}
	return cfg
}
