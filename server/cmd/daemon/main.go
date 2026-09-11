// Command daemon runs on the owner's machine and is the only thing that
// touches an agent CLI.
//
// It dials out to the server and nothing ever dials in, so the machine needs no
// open port, no forwarding and no inbound firewall rule (AGENTS.md §3).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/agent"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	serverURL := env("SERVER_WS", "ws://localhost:8080/daemon")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	d := &daemon{log: log, backends: agent.Backends()}

	// Bounded exponential backoff with full jitter. The attempt counter resets
	// only after a connection has actually been established, not after a dial
	// returned — a socket that drops immediately is still a failure, and
	// resetting on the dial is what turns a flapping link into a hot loop.
	attempt := 0
	for ctx.Err() == nil {
		if err := d.run(ctx, serverURL); err != nil && ctx.Err() == nil {
			log.Warn("connection lost", "err", err, "attempt", attempt+1)
		}
		if ctx.Err() != nil {
			break
		}
		if d.connectedOnce {
			attempt = 0
			d.connectedOnce = false
		}
		delay := backoff(attempt)
		attempt++
		log.Info("reconnecting", "in", delay)
		select {
		case <-ctx.Done():
		case <-time.After(delay):
		}
	}
	log.Info("daemon stopped")
}

const (
	backoffBase = 500 * time.Millisecond
	backoffMax  = 30 * time.Second
)

// backoff returns a delay in [0, min(base<<attempt, max)]. Full jitter, so a
// fleet of machines waking from sleep together does not retry in lockstep.
func backoff(attempt int) time.Duration {
	d := backoffBase << min(attempt, 16)
	if d > backoffMax {
		d = backoffMax
	}
	return time.Duration(rand.Int63n(int64(d) + 1))
}

type daemon struct {
	log      *slog.Logger
	backends map[string]agent.Backend

	mu   sync.Mutex
	conn *websocket.Conn

	// connectedOnce records that this attempt got as far as a working socket,
	// which is what makes resetting the retry counter meaningful.
	connectedOnce bool
}

func (d *daemon) run(ctx context.Context, url string) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	d.mu.Lock()
	d.conn = conn
	d.mu.Unlock()
	d.connectedOnce = true
	d.log.Info("connected", "server", url)

	// Report what is installed before anything can be asked of us, so the
	// surface never offers a provider this machine cannot run.
	providers := agent.Detect(ctx)
	for _, p := range providers {
		d.log.Info("provider", "id", p.ID, "availability", p.Availability, "models", len(p.Models))
	}
	if err := d.send(protocol.DaemonMessage{
		Type: protocol.DaemonHello, Machine: hostname(), Providers: providers,
	}); err != nil {
		return err
	}

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var msg protocol.ServerMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			d.log.Warn("bad server message", "err", err)
			continue
		}
		switch {
		case msg.Type == protocol.ServerRunTurn && msg.Turn != nil:
			// Each turn runs on its own goroutine so a long one does not block
			// the socket, and cancelling the daemon cancels the CLI with it.
			go d.runTurn(ctx, *msg.Turn)
		case msg.Type == protocol.ServerListDir:
			// Off the read loop too: a directory on a cold or network drive can
			// take a moment, and a turn already streaming must not stall behind
			// somebody browsing for a folder.
			go d.listDir(msg)
		}
	}
}

func (d *daemon) send(msg protocol.DaemonMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.conn == nil {
		return fmt.Errorf("not connected")
	}
	return d.conn.WriteMessage(websocket.TextMessage, payload)
}

// listDir answers a browser asking what is on this machine. Only the daemon can
// see the filesystem — the server is meant to run somewhere else — so every path
// question comes through here.
func (d *daemon) listDir(msg protocol.ServerMessage) {
	listing := agent.Listing(msg.Path)
	if err := d.send(protocol.DaemonMessage{
		Type:      protocol.DaemonDirListing,
		RequestID: msg.RequestID,
		Listing:   &listing,
	}); err != nil {
		d.log.Warn("dir listing not sent", "err", err, "path", msg.Path)
	}
}

func (d *daemon) runTurn(ctx context.Context, t protocol.RunTurn) {
	log := d.log.With("turn", t.TurnID, "provider", t.Provider, "model", t.Model.ID)

	backend, ok := d.backends[t.Provider]
	if !ok {
		_ = d.send(protocol.DaemonMessage{
			Type: protocol.DaemonFailed, TurnID: t.TurnID,
			Error: fmt.Sprintf("no adapter for provider %q", t.Provider),
		})
		return
	}

	prompt := buildPrompt(t)
	log.Info("running turn", "replay", len(t.Replay), "resume", t.ResumeSessionID != "")

	session, err := backend.Execute(ctx, prompt, agent.ExecOptions{
		Cwd:             t.Cwd,
		Model:           t.Model.ID,
		ResumeSessionID: t.ResumeSessionID,
	})
	if err != nil {
		_ = d.send(protocol.DaemonMessage{
			Type: protocol.DaemonFailed, TurnID: t.TurnID, Error: err.Error(),
		})
		return
	}

	for msg := range session.Messages {
		switch msg.Type {
		case agent.MessageStarted:
			_ = d.send(protocol.DaemonMessage{
				Type: protocol.DaemonStarted, TurnID: t.TurnID, SessionID: msg.SessionID,
			})
		case agent.MessageDelta:
			_ = d.send(protocol.DaemonMessage{
				Type: protocol.DaemonDelta, TurnID: t.TurnID, Text: msg.Text,
			})
		case agent.MessageLimit:
			_ = d.send(protocol.DaemonMessage{
				Type: protocol.DaemonLimit, TurnID: t.TurnID,
				Provider: t.Provider, Headroom: msg.Headroom,
			})
		}
	}

	result := <-session.Result
	if result.Err != nil {
		log.Warn("turn failed", "err", result.Err)
		_ = d.send(protocol.DaemonMessage{
			Type: protocol.DaemonFailed, TurnID: t.TurnID, Error: result.Err.Error(),
		})
		return
	}
	log.Info("turn done", "tokens", result.Tokens, "chars", len(result.Text))
	_ = d.send(protocol.DaemonMessage{
		Type: protocol.DaemonDone, TurnID: t.TurnID,
		Full: result.Text, Tokens: result.Tokens, SpendTicks: result.SpendTicks,
		SessionID: result.SessionID,
	})
}

// buildPrompt prepends the catch-up, when there is one.
//
// This is what "moving a conversation" physically is: no CLI can resume
// another's session, so a provider that has not seen the history is told it in
// the prompt. The framing is explicit rather than disguised as dialogue —
// pretending the other agent's words were its own would make it answer as if it
// had already committed to them.
func buildPrompt(t protocol.RunTurn) string {
	if len(t.Replay) == 0 {
		return t.Prompt
	}
	var b strings.Builder
	b.WriteString("You are joining a conversation already in progress. ")
	b.WriteString("Earlier turns are below, some answered by a different coding agent. ")
	b.WriteString("Read them for context, then answer the final message.\n\n")
	b.WriteString("--- conversation so far ---\n")
	for _, e := range t.Replay {
		switch e.Role {
		case "user":
			b.WriteString("\n[user]\n")
		case "agent":
			fmt.Fprintf(&b, "\n[%s]\n", e.Provider)
		default:
			continue
		}
		b.WriteString(strings.TrimSpace(e.Text))
		b.WriteString("\n")
	}
	b.WriteString("--- end of conversation so far ---\n\n")
	b.WriteString("[user]\n")
	b.WriteString(t.Prompt)
	return b.String()
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
