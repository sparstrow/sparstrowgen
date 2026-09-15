// Command daemon runs on the owner's machine and is the only thing that
// touches an agent CLI.
//
// It dials out to the server and nothing ever dials in, so the machine needs no
// open port, no forwarding and no inbound firewall rule (AGENTS.md §3).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/agent"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

func main() {
	args := os.Args[1:]
	switch {
	case len(args) == 1 && isPairingLink(args[0]):
		os.Exit(report(activate(args[0]), ""))
	case len(args) > 0 && args[0] == "install":
		err := install()
		os.Exit(report(err, installedNotice()))
	case len(args) > 0 && args[0] == "pair":
		os.Exit(pairCommand(args[1:]))
	case len(args) > 0 && args[0] == "run":
		runBackground()
	case len(args) > 0 && args[0] == "apply-update":
		// The updater process, started by a copy handing over (update.go).
		os.Exit(applyUpdateCommand(args[1:]))
	case len(args) == 0 && released():
		// Double-clicking sparstrowgen-setup.exe.
		err := install()
		os.Exit(report(err, installedNotice()))
	case len(args) == 0:
		runForeground()
	default:
		fmt.Fprintln(os.Stderr, "usage: daemon [run | install | pair -request <id> | sparstrowgen://pair?request=<id>]")
		os.Exit(2)
	}
}

// installedNotice is what a finished install says. A computer that is already
// paired needs nothing more, so it is not sent back to Add computer (B-27).
func installedNotice() string {
	if readCredential(credentialName) != "" {
		return "sparstrowgen is updated on this computer.\n\nThis computer is already paired, so it reconnects by itself. There is nothing else to do."
	}
	return "sparstrowgen is installed on this computer and will start when you sign in to Windows.\n\nGo back to sparstrowgen in your browser, open Machines and choose Add computer."
}

// report shows the outcome to the person who opened the executable and returns
// its exit code. A successful pairing link says nothing: the browser moves on.
func report(err error, success string) int {
	if err != nil {
		notify("sparstrowgen could not finish.\n\n"+err.Error(), true)
		return 1
	}
	if success != "" {
		notify(success, false)
	}
	return 0
}

// runForeground is `go run ./cmd/daemon`: logs in the terminal, Ctrl+C stops it.
func runForeground() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if !serve(ctx, log) {
		os.Exit(1)
	}
}

// runBackground is the installed daemon: no window, logs in a file, and at
// most one copy per Windows user.
func runBackground() {
	log, closeLog := fileLogger()
	defer closeLog()
	release, ok := singleInstance()
	if !ok {
		log.Info("sparstrowgen is already running for this Windows user")
		return
	}
	defer release()
	ensureHiddenConsole()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	watchForExit(stop)
	serve(ctx, log)
}

// fileLogger writes to <data>/logs/daemon.log, keeping one previous file once
// it passes 5 MB. It discards when there is nowhere to write.
func fileLogger() (*slog.Logger, func()) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	dir, err := dataDir()
	if err != nil {
		return discard, func() {}
	}
	logs := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logs, 0o700); err != nil {
		return discard, func() {}
	}
	path := filepath.Join(logs, "daemon.log")
	if info, err := os.Stat(path); err == nil && info.Size() > 5<<20 {
		_ = os.Rename(path, path+".1")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return discard, func() {}
	}
	return slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo})), func() { _ = f.Close() }
}

// serve connects and reconnects until ctx ends or the computer is disconnected.
// It reports false when it could not start at all.
func serve(ctx context.Context, log *slog.Logger) bool {
	// The agent package reports a degraded process-tree stop through the
	// default logger rather than by threading one through three backends.
	slog.SetDefault(log)
	url := serverWS()
	// Handing over to an update ends this copy the same way a stop request does.
	ctx, handOver := context.WithCancel(ctx)
	defer handOver()
	d := &daemon{log: log, backends: agent.Backends(), turns: newRunningTurns()}
	d.updates = newUpdater(log, d.turns, d.send, handOver)
	if d.updates != nil {
		go d.updates.loop(ctx, d.isConnected)
	}

	// Bounded exponential backoff with full jitter. The attempt counter resets
	// only after a connection has actually been established, not after a dial
	// returned — a socket that drops immediately is still a failure, and
	// resetting on the dial is what turns a flapping link into a hot loop.
	attempt := 0
	for ctx.Err() == nil {
		// Read on every attempt: pairing writes a new credential while this
		// copy may already be waiting.
		token, kind := machineToken()
		if token == "" {
			if released() {
				log.Info("this computer is not paired yet; pair it from Machines in sparstrowgen")
			} else {
				log.Error("no machine credential is available — pair this computer or set DAEMON_TOKEN for the legacy local route")
			}
			return false
		}
		d.onConnect = nil
		if kind == pendingCredential {
			d.onConnect = func() {
				if err := promotePending(token); err != nil {
					log.Error("could not keep the newly approved credential", "err", err)
					return
				}
				log.Info("this computer's pairing was approved")
			}
		}
		err := d.run(ctx, url, token)
		if errors.Is(err, errDisconnected) {
			if kind == pendingCredential {
				// Declined or expired before approval. Any earlier pairing is
				// untouched, so go straight back to it.
				log.Warn("this pairing was declined or has expired; keeping any earlier pairing")
				if err := forgetCredential(pendingName, token); err != nil {
					log.Error("could not remove the refused pairing", "err", err)
					return true
				}
				attempt = 0
				continue
			}
			log.Warn("this computer was disconnected from its account; forgetting its credential")
			if kind == pairedCredential {
				if err := forgetCredential(credentialName, token); err != nil {
					log.Error("could not remove the revoked credential", "err", err)
				}
			}
			return true
		}
		if err != nil && ctx.Err() == nil {
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
		if errors.Is(err, errAwaitingApproval) && kind == pendingCredential && recentlyPaired() {
			delay = approvalPoll
		}
		log.Info("reconnecting", "in", delay)
		select {
		case <-ctx.Done():
		case <-time.After(delay):
		}
	}
	log.Info("daemon stopped")
	return true
}

// approvalPoll is how often a freshly paired computer checks whether it has
// been approved, so the browser is not left waiting on a 30s backoff.
const approvalPoll = 2 * time.Second

var (
	errAwaitingApproval = errors.New("the server has not accepted this computer: it is waiting for approval in the browser, or its credential does not match")
	errDisconnected     = errors.New("this computer was disconnected from its account")
)

const (
	backoffBase = 500 * time.Millisecond
	backoffMax  = 30 * time.Second
)

// idleBudget is how long a turn may produce NOTHING before it is ended.
//
// Not a total timeout. Multica learned that one the expensive way (MUL-3064):
// a wall-clock cap kills a session that is working perfectly well and merely
// taking a while, which on a coding agent is most real tasks. Liveness is a
// question about silence, not about duration, so a turn streaming for an hour
// is fine and a turn silent for fifteen minutes is not.
//
// **Deliberately generous, and the asymmetry is the reason.** A false positive
// throws away real work and the quota spent earning it; a false negative just
// means waiting a bit longer for a safety net that only exists for when nobody
// is watching — because since the stop button shipped, a person who IS watching
// can end a turn themselves in one click.
//
// The budget is tightest on codex, which emits nothing at all between its
// session id and its finished answer, so for codex this is effectively a cap on
// the whole turn rather than a silence detector. That is the honest boundary of
// what the CLI tells us, and it is why fifteen minutes rather than three.
var idleBudget = envDuration("TURN_IDLE_TIMEOUT", 15*time.Minute)

// envDuration reads a Go duration like "20m" or "90s". A value that does not
// parse is worth saying out loud rather than silently ignoring: someone setting
// it meant something by it, and falling back without a word would hide that the
// setting did nothing.
func envDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		slog.Warn("ignoring an unusable "+key, "value", raw, "using", fallback)
		return fallback
	}
	return d
}

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
	turns    *runningTurns

	mu   sync.Mutex
	conn *websocket.Conn

	// connectedOnce records that this attempt got as far as a working socket,
	// which is what makes resetting the retry counter meaningful.
	connectedOnce bool

	// onConnect runs once a dial succeeds. A pending credential becomes this
	// computer's credential only here, when the server has accepted it.
	onConnect func()

	// detect reports the installed providers; nil means agent.Detect. Tests
	// replace it so a connection never runs the real CLIs.
	detect func(context.Context) []protocol.Provider

	// updates is nil when this copy cannot update itself (update.go).
	updates *updater
}

func (d *daemon) run(ctx context.Context, url, token string) error {
	// In a header rather than the URL: a query string ends up in proxy logs and
	// in the server's own access log, and a credential that is written down
	// somewhere by default is a credential that leaks eventually.
	conn, res, err := websocket.DefaultDialer.DialContext(ctx, url, http.Header{
		"Authorization": []string{"Bearer " + token},
	})
	if err != nil {
		// 401 and 403 are not connection problems, so say what they are rather
		// than letting them look like the network being down.
		if res != nil {
			switch res.StatusCode {
			case http.StatusUnauthorized:
				return errAwaitingApproval
			case http.StatusForbidden:
				return errDisconnected
			}
		}
		return err
	}
	defer conn.Close()

	// Stopping cancels ctx, but the read below does not watch ctx, so a copy
	// that was connected never noticed it had been asked to exit and the
	// installer gave up waiting for it (docs/Bugs.md B-26). Closing the socket
	// ends the read.
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "stopping"), time.Now().Add(time.Second))
			_ = conn.Close()
		case <-finished:
		}
	}()

	d.mu.Lock()
	d.conn = conn
	d.mu.Unlock()
	d.connectedOnce = true
	d.log.Info("connected", "server", url)
	if d.onConnect != nil {
		d.onConnect()
	}

	// Nothing can be delivered once this socket is gone, and the server gives up
	// on turns it can no longer hear about (docs/Bugs.md B-8). A CLI still
	// running past that point is spending the owner's quota on an answer that
	// has nowhere to go.
	defer func() {
		if n := d.turns.cancelAll(); n > 0 {
			d.log.Warn("cancelled turns with nowhere to report", "turns", n)
		}
	}()

	// Report what is installed before anything can be asked of us, so the
	// surface never offers a provider this machine cannot run.
	detect := d.detect
	if detect == nil {
		detect = agent.Detect
	}
	providers := detect(ctx)
	for _, p := range providers {
		d.log.Info("provider", "id", p.ID, "availability", p.Availability, "models", len(p.Models))
	}
	// Cleared on the way out, so nothing reads a closed socket as connected.
	defer func() {
		d.mu.Lock()
		if d.conn == conn {
			d.conn = nil
		}
		d.mu.Unlock()
	}()
	if err := d.send(protocol.DaemonMessage{
		Type: protocol.DaemonHello, Machine: hostname(), Providers: providers,
		Version: version, Protocol: protocol.DaemonProtocol, SelfUpdates: d.updates != nil,
	}); err != nil {
		return err
	}
	if d.updates != nil {
		d.updates.connected()
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
		case msg.Type == protocol.ServerStopTurn && msg.TurnID != "":
			// Deliberately NOT on a goroutine. Stopping is a map write and a
			// context cancel with no I/O in it, and running it on the read loop
			// is what keeps it ordered against the run_turn that started it.
			if !d.turns.stop(msg.TurnID) {
				// Either it finished a moment ago or its start is still in
				// flight; stop() has remembered the request either way.
				d.log.Info("stop for a turn that is not running", "turn", msg.TurnID)
			}
		case msg.Type == protocol.ServerListDir:
			// Off the read loop too: a directory on a cold or network drive can
			// take a moment, and a turn already streaming must not stall behind
			// somebody browsing for a folder.
			go d.listDir(msg)
		case msg.Type == protocol.ServerCheckUpdate || msg.Type == protocol.ServerApplyUpdate:
			// Off the read loop: a check can go on to download an installer.
			go d.answerUpdate(ctx, msg)
		case msg.Type == protocol.ServerUpdatePreference && msg.Automatic != nil:
			if d.updates != nil {
				d.updates.setAutomatic(*msg.Automatic)
			}
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
	// Windows refuses to start a process in a missing folder, and says so by
	// naming the executable (B-30), which reads as a broken agent install.
	if problem := folderProblem(t.Cwd); problem != "" {
		log.Info("turn refused: conversation folder unusable", "cwd", t.Cwd)
		_ = d.send(protocol.DaemonMessage{Type: protocol.DaemonFailed, TurnID: t.TurnID, Error: problem})
		return
	}

	// Per-turn cancellation, so stopping one turn leaves the others alone. The
	// connection context stays the parent: losing the socket still stops
	// everything, which is the behaviour that existed before this.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if !d.turns.begin(t.TurnID, cancel) {
		if d.turns.closedForUpdate() {
			// Arrived in the moment between the last idle check and handing over.
			log.Info("turn refused: this copy is installing an update")
			_ = d.send(protocol.DaemonMessage{
				Type: protocol.DaemonFailed, TurnID: t.TurnID,
				Error: "this computer is installing a sparstrowgen update — send the message again in a moment",
			})
			return
		}
		// The stop overtook the start. Nothing ran, so there is nothing to kill
		// and no text to keep — but the turn still has to be closed out, or the
		// server waits on it forever.
		log.Info("turn was stopped before it started")
		_ = d.send(protocol.DaemonMessage{Type: protocol.DaemonStopped, TurnID: t.TurnID})
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
		if d.turns.end(t.TurnID) {
			_ = d.send(protocol.DaemonMessage{Type: protocol.DaemonStopped, TurnID: t.TurnID})
			return
		}
		_ = d.send(protocol.DaemonMessage{
			Type: protocol.DaemonFailed, TurnID: t.TurnID, Error: err.Error(),
		})
		return
	}

	// The watchdog. A CLI that wedges produces nothing and never exits, and
	// before this the turn simply never ended (docs/Bugs.md B-8). Every message
	// is a sign of life and resets the budget.
	idle := time.NewTimer(idleBudget)
	defer idle.Stop()
	wentQuiet := false

	drain := true
	for drain {
		select {
		case msg, ok := <-session.Messages:
			if !ok {
				drain = false
				break
			}
			idle.Reset(idleBudget)
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

		case <-idle.C:
			// Cancel, but keep draining. Messages is unbuffered past its
			// capacity and the parser blocks writing to it, so abandoning the
			// loop here would wedge the very goroutine we are trying to stop.
			wentQuiet = true
			log.Warn("agent went quiet; stopping it", "after", idleBudget)
			cancel()
		}
	}

	result := <-session.Result

	// One place decides how the turn ended, and it reads the stop flag exactly
	// once. A killed CLI usually also reports an error on its way out, so this
	// order matters: asked-to-stop wins over whatever the dying process said,
	// because that error is a consequence of the stop and not a reason.
	switch stopped := d.turns.end(t.TurnID); {
	case stopped: // deliberate, and it outranks everything below
		// Text that arrived before the stop is kept, and it is usually the
		// point — a turn is stopped once the answer has gone somewhere
		// useless, and what came before is still worth reading. A provider
		// that does not stream (codex) simply has nothing to keep.
		log.Info("turn stopped", "chars", len(result.Text))
		_ = d.send(protocol.DaemonMessage{
			Type: protocol.DaemonStopped, TurnID: t.TurnID,
			Full: result.Text, Tokens: result.Tokens, SpendTicks: result.SpendTicks,
			SessionID: result.SessionID,
		})
	case wentQuiet:
		// Ahead of result.Err for the same reason `stopped` is: the error a
		// killed CLI reports on its way out is a consequence of the kill, and
		// "it stopped saying anything" is the truer account of what happened.
		log.Warn("turn abandoned after silence", "after", idleBudget, "chars", len(result.Text))
		_ = d.send(protocol.DaemonMessage{
			Type: protocol.DaemonFailed, TurnID: t.TurnID,
			Full: result.Text,
			Error: fmt.Sprintf("%s stopped responding — nothing arrived for %s, so it was ended",
				t.Provider, idleBudget),
		})
	case result.Err != nil:
		log.Warn("turn failed", "err", result.Err)
		// Full, not just the error: the parser may have recovered whole messages
		// before the turn died, and on a provider that does not stream those are
		// the ONLY copy — the server has no deltas to fall back on (B-10).
		_ = d.send(protocol.DaemonMessage{
			Type: protocol.DaemonFailed, TurnID: t.TurnID,
			Full: result.Text, Error: result.Err.Error(),
		})
	default:
		log.Info("turn done", "tokens", result.Tokens, "chars", len(result.Text))
		_ = d.send(protocol.DaemonMessage{
			Type: protocol.DaemonDone, TurnID: t.TurnID,
			Full: result.Text, Tokens: result.Tokens, SpendTicks: result.SpendTicks,
			SessionID: result.SessionID,
		})
	}
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
