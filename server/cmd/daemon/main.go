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
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
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
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	// The agent package reports a degraded process-tree stop through the
	// default logger rather than by threading one through three backends.
	slog.SetDefault(log)
	executable, err := os.Executable()
	if err != nil {
		log.Error("could not locate the daemon executable", "err", err)
		os.Exit(1)
	}
	config, err := loadInstallationConfig(executable)
	if err != nil {
		log.Error("could not read the installed connection configuration", "err", err)
		os.Exit(1)
	}
	serverURL := serverWS(config)
	if len(os.Args) > 1 && os.Args[1] == "pair" {
		if err := pairMain(config, serverURL); err != nil {
			log.Error("could not claim pairing request", "err", err)
			os.Exit(1)
		}
		return
	}

	// The shared secret proving this is the machine that was installed, not
	// something else that found the socket. Required, for the same reason the
	// server requires it: a daemon that silently connects without one would be
	// refused by the server anyway, and failing here says why in one line
	// instead of as an endless reconnect loop.
	token := pairedCredential()
	if token == "" {
		token = os.Getenv("DAEMON_TOKEN")
	}
	if token == "" {
		log.Error("no machine credential is available — pair this computer or set DAEMON_TOKEN for the legacy local route")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	d := &daemon{log: log, backends: agent.Backends(), turns: newRunningTurns()}

	// Bounded exponential backoff with full jitter. The attempt counter resets
	// only after a connection has actually been established, not after a dial
	// returned — a socket that drops immediately is still a failure, and
	// resetting on the dial is what turns a flapping link into a hot loop.
	attempt := 0
	for ctx.Err() == nil {
		if err := d.run(ctx, serverURL, token); err != nil && ctx.Err() == nil {
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
}

type installationConfig struct {
	ServerAPI string `json:"serverApi"`
	ServerWS  string `json:"serverWs"`
}

func loadInstallationConfig(executable string) (installationConfig, error) {
	path := filepath.Join(filepath.Dir(executable), "sparstrowgen.json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return installationConfig{}, nil
	}
	if err != nil {
		return installationConfig{}, fmt.Errorf("read installation configuration: %w", err)
	}
	var config installationConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return installationConfig{}, fmt.Errorf("read installation configuration: %w", err)
	}
	if config.ServerAPI != "" {
		u, err := url.Parse(config.ServerAPI)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return installationConfig{}, errors.New("installation configuration has an invalid serverApi")
		}
	}
	if config.ServerWS != "" {
		u, err := url.Parse(config.ServerWS)
		if err != nil || (u.Scheme != "ws" && u.Scheme != "wss") || u.Host == "" {
			return installationConfig{}, errors.New("installation configuration has an invalid serverWs")
		}
	}
	return config, nil
}

func serverWS(config installationConfig) string {
	if configured := os.Getenv("SERVER_WS"); configured != "" {
		return configured
	}
	if config.ServerWS != "" {
		return config.ServerWS
	}
	return "ws://localhost:8080/daemon"
}

func serverAPI(config installationConfig, ws string) string {
	if configured := os.Getenv("SERVER_API"); configured != "" {
		return configured
	}
	if config.ServerAPI != "" {
		return config.ServerAPI
	}
	return strings.Replace(strings.Replace(ws, "ws://", "http://", 1), "wss://", "https://", 1)
}

func (d *daemon) run(ctx context.Context, url, token string) error {
	// In a header rather than the URL: a query string ends up in proxy logs and
	// in the server's own access log, and a credential that is written down
	// somewhere by default is a credential that leaks eventually.
	conn, res, err := websocket.DefaultDialer.DialContext(ctx, url, http.Header{
		"Authorization": []string{"Bearer " + token},
	})
	if err != nil {
		// 401 is not a connection problem and will not fix itself by retrying,
		// so say what it actually is rather than letting it look like the
		// network being down.
		if res != nil && res.StatusCode == http.StatusUnauthorized {
			return errors.New("the server rejected this machine: its credential was revoked or it is still waiting for approval")
		}
		return err
	}
	defer conn.Close()

	d.mu.Lock()
	d.conn = conn
	d.mu.Unlock()
	d.connectedOnce = true
	d.log.Info("connected", "server", url)

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
		}
	}
}

func credentialPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "sparstrowgen", "machine-credential")
}
func pairedCredential() string {
	if raw := os.Getenv("SPARSTROWGEN_MACHINE_CREDENTIAL"); raw != "" {
		return raw
	}
	p := credentialPath()
	if p == "" {
		return ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
func pairMain(config installationConfig, ws string) error {
	fs := flag.NewFlagSet("pair", flag.ExitOnError)
	request := fs.String("request", "", "one-time pairing request")
	name := fs.String("name", env("COMPUTERNAME", "This computer"), "computer name")
	fs.Parse(os.Args[2:])
	if *request == "" {
		return errors.New("pair requires -request from the sparstrowgen launch link")
	}
	base := serverAPI(config, ws)
	base = strings.TrimSuffix(base, "/daemon")
	body := strings.NewReader(fmt.Sprintf(`{"request":%q,"name":%q}`, *request, *name))
	res, err := http.Post(base+"/daemon/pair", "application/json", body)
	if err != nil {
		return fmt.Errorf("call pairing service: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("pairing was refused: %s", strings.TrimSpace(string(raw)))
	}
	var reply struct {
		Credential string `json:"credential"`
	}
	if err := json.NewDecoder(res.Body).Decode(&reply); err != nil {
		return fmt.Errorf("pairing response had no credential: %w", err)
	}
	if reply.Credential == "" {
		return errors.New("pairing response had no credential")
	}
	p := credentialPath()
	if p == "" {
		return errors.New("could not choose a secure credential location")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return fmt.Errorf("prepare credential location: %w", err)
	}
	if err := os.WriteFile(p, []byte(reply.Credential+"\n"), 0600); err != nil {
		return fmt.Errorf("save machine credential: %w", err)
	}
	return nil
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

	// Per-turn cancellation, so stopping one turn leaves the others alone. The
	// connection context stays the parent: losing the socket still stops
	// everything, which is the behaviour that existed before this.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if !d.turns.begin(t.TurnID, cancel) {
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
