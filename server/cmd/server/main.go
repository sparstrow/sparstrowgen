// Command server is the web-facing half: HTTP for everything the browser does,
// a websocket for what it watches, and one websocket the daemon dials in on.
//
// It never reaches out to the owner's machine. Nothing inbound to that machine
// exists at all — the daemon dials out (AGENTS.md §3).
package main

import (
	"context"
	"errors"
	"log/slog"
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

	dsn := env("DATABASE_URL",
		"postgres://sparstrowgen:sparstrowgen@localhost:5433/sparstrowgen?sslmode=disable")
	addr := env("ADDR", ":8080")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Error("connect to postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Error("postgres unreachable", "dsn", dsn, "err", err)
		os.Exit(1)
	}

	h := hub.New(log)
	srv := &http.Server{
		Addr:    addr,
		Handler: api.New(store.New(pool), h, log).Routes(),
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

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
