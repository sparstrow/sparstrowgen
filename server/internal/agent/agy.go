package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// Agy drives the `agy` CLI (Antigravity).
//
// Two things make it different from the other two. Its stream keys events on
// "event" rather than "type". And it is a ROUTER, not a vendor: `agy models`
// returns Gemini, Claude and GPT-OSS models behind one CLI, with the reasoning
// effort baked into the model id (gemini-3.1-pro-high vs -low) rather than
// offered as a separate setting.
//
// Several of its quirks were found first by Multica, which drives the same CLI
// (Reference/multica-main/server/pkg/agent/antigravity.go): the five-minute
// print timeout, the silent no-op on an unknown model, and failures that exit 0.
type Agy struct{}

func (Agy) ID() string { return "agy" }

// agyPrintTimeout replaces agy's own five-minute limit, which has no "off"
// value (docs/Bugs.md B-33). It is deliberately far past the daemon's silence
// watchdog, which is what decides that a turn is stuck.
const agyPrintTimeout = 24 * time.Hour

func agyArgs(opts ExecOptions, logPath string) []string {
	args := []string{
		// The prompt arrives on stdin as one stream-json message, never on the
		// command line, which Windows caps at 32,767 characters (B-32). agy still
		// wants -p, with its value attached: a bare -p takes the next flag as the
		// prompt.
		"-p=",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--print-timeout", agyPrintTimeout.String(),
	}
	if logPath != "" {
		// Some failures appear only here, with exit code 0 (agyLogFailure).
		args = append(args, "--log-file", logPath)
	}
	if opts.Cwd != "" {
		// Running in the folder is not enough. agy reports the right cwd in
		// its init event and then resolves "notes.txt" against its own scratch
		// workspace, so it never saw the project a conversation is about
		// (docs/Bugs.md B-55). Adding the folder to its workspace is what makes
		// it look there — verified on agy 1.2.3.
		args = append(args, "--add-dir", opts.Cwd)
	}
	// The conversation's files folder, so it can read what was sent with a
	// message. Verified 2026-09-24 on 1.2.10: a second --add-dir is read.
	for _, dir := range opts.AddDirs {
		args = append(args, "--add-dir", dir)
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.ResumeSessionID != "" {
		args = append(args, "--conversation", opts.ResumeSessionID)
	}
	return args
}

// agyInput is the one stream-json message a turn sends on stdin. Verified
// against agy 1.2.3: it keys on "event" like its output does.
func agyInput(prompt string) ([]byte, error) {
	data, err := json.Marshal(map[string]any{
		"event":   "user",
		"message": map[string]any{"role": "user", "content": prompt},
	})
	if err != nil {
		return nil, fmt.Errorf("encode the prompt for agy: %w", err)
	}
	return append(data, '\n'), nil
}

func (a Agy) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	if err := agyModelError(opts.Model, agyCatalog.get(ctx)); err != nil {
		return nil, err
	}
	input, err := agyInput(prompt)
	if err != nil {
		return nil, err
	}
	logPath := ""
	if f, err := os.CreateTemp("", "sparstrowgen-agy-*.log"); err == nil {
		logPath = f.Name()
		_ = f.Close()
	}
	removeLog := func() {
		if logPath != "" {
			_ = os.Remove(logPath)
		}
	}

	cmd := command(ctx, opts.Cwd, "agy", agyArgs(opts, logPath)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		removeLog()
		return nil, err
	}
	cmd.Stdin = bytes.NewReader(input)
	messages := make(chan Message, 64)
	rec := record(cmd, messages)
	// launch rather than cmd.Start: a stop has to take the tool subprocesses
	// with it, not just the CLI (D-021).
	proc, err := launch(ctx, cmd, stdout)
	if err != nil {
		removeLog()
		return nil, err
	}

	result := make(chan Result, 1)

	go func() {
		defer close(result)
		defer removeLog()
		p := parseAgy(rec.stdout(stdout), messages)

		// Reaped before Messages closes: stderr keeps arriving until Wait
		// returns, and every line of it belongs in the record.
		waitErr := proc.Wait()
		rec.finish()
		close(messages)
		switch {
		case p.Err != nil:
			waitErr = p.Err
		case p.Text == "" && ctx.Err() == nil:
			// agy can end a turn with nothing to show and still report success.
			// A blank "done" reads as the app losing the answer.
			waitErr = agyQuietFailure(readSmall(logPath), waitErr)
		}
		result <- Result{
			Text:      p.Text,
			SessionID: p.SessionID,
			Tokens:    p.Tokens,
			// Tokens only — agy states no currency figure.
			Err: waitErr,
		}
	}()

	return &Session{Messages: messages, Result: result, Sent: sentBy(cmd, opts, prompt, input)}, nil
}

// ---------------------------------------------------------------------------
// the model catalogue
// ---------------------------------------------------------------------------

// agyCatalogTTL bounds how stale a remembered `agy models` answer may be. The
// command takes about two seconds, too slow to run before every turn.
const agyCatalogTTL = 10 * time.Minute

type agyCatalogCache struct {
	mu     sync.Mutex
	at     time.Time
	models []protocol.Model
}

var agyCatalog = &agyCatalogCache{}

func (c *agyCatalogCache) put(models []protocol.Model) {
	if len(models) == 0 {
		return // a failed listing must not erase a good one
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at, c.models = time.Now(), models
}

func (c *agyCatalogCache) get(ctx context.Context) []protocol.Model {
	c.mu.Lock()
	fresh := !c.at.IsZero() && time.Since(c.at) < agyCatalogTTL
	models := c.models
	c.mu.Unlock()
	if fresh {
		return models
	}
	if listed := agyModels(ctx); len(listed) > 0 {
		c.put(listed)
		return listed
	}
	return models
}

// agyModelError refuses a model this agy does not list. agy itself exits 0 with
// no output on an unknown model, which would show as an empty answer. With no
// catalogue at all it lets agy decide, so a listing hiccup never blocks a turn.
func agyModelError(model string, catalog []protocol.Model) error {
	if model == "" || len(catalog) == 0 {
		return nil
	}
	names := make([]string, 0, len(catalog))
	for _, m := range catalog {
		if m.ID == model {
			return nil
		}
		names = append(names, m.ID)
	}
	return fmt.Errorf("agy on this computer does not offer the model %q. Choose another agy model: %s",
		model, strings.Join(names, ", "))
}

// ---------------------------------------------------------------------------
// failures that exit 0
// ---------------------------------------------------------------------------

var (
	// Written when agy's own print timeout elapses, after which it prints an
	// error and exits 0 (Multica MUL-3570).
	agyTimedOutRe = regexp.MustCompile(`Print mode: timed out after \d+ polls`)
	// Written for a model or provider failure agy does not otherwise report.
	agyExecutorErrorRe = regexp.MustCompile(`agent executor error:\s*(.+)`)
)

// agyQuietFailure explains a turn that produced no answer, from agy's log.
func agyQuietFailure(log string, waitErr error) error {
	if m := agyExecutorErrorRe.FindAllStringSubmatch(log, -1); len(m) > 0 {
		return fmt.Errorf("agy failed: %s", strings.TrimSpace(m[len(m)-1][1]))
	}
	if agyTimedOutRe.MatchString(log) {
		return errors.New("agy gave up waiting for its own answer")
	}
	if waitErr != nil {
		return fmt.Errorf("agy exited without answering: %w", waitErr)
	}
	return errors.New("agy finished without writing an answer")
}

// readSmall reads a log for failure markers, keeping only its last megabyte.
func readSmall(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if len(data) > 1<<20 {
		data = data[len(data)-1<<20:]
	}
	return string(data)
}

// ---------------------------------------------------------------------------
// the stream
// ---------------------------------------------------------------------------

func parseAgy(r io.Reader, out chan<- Message) parsed {
	var (
		p        parsed
		streamed strings.Builder
		final    string
		status   string
		failure  string
		denied   []string
	)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		var ev agyEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		switch ev.Event {
		case "init":
			if ev.ConversationID != "" {
				p.SessionID = ev.ConversationID
				send(out, Message{Type: MessageStarted, SessionID: p.SessionID})
			}
		case "step_update":
			if ev.StepUpdate.TextDelta != "" {
				streamed.WriteString(ev.StepUpdate.TextDelta)
				send(out, Message{Type: MessageDelta, Text: ev.StepUpdate.TextDelta})
			}
		case "result":
			status = ev.Result.Status
			failure = ev.Result.Error
			p.Tokens = ev.Result.Usage.total()
			// result.response is authoritative: it is the whole answer, whether
			// or not deltas arrived.
			final = ev.Result.Response
			if ev.Result.ConversationID != "" {
				p.SessionID = ev.Result.ConversationID
			}
			for _, d := range ev.Result.DeniedActions {
				if d.DisplayName != "" {
					denied = append(denied, d.DisplayName)
				} else if d.Action != "" {
					denied = append(denied, d.Action)
				}
			}
		}
	}

	p.Text = strings.TrimRight(final, "\n")
	if p.Text == "" {
		p.Text = streamed.String()
	}
	switch {
	case status != "" && status != "SUCCESS":
		if failure != "" {
			p.Err = fmt.Errorf("agy failed: %s", failure)
		} else {
			p.Err = fmt.Errorf("agy finished with status %s", status)
		}
	case p.Text == "" && len(denied) > 0:
		// Seen 2026-09-14 on agy 1.2.3: status SUCCESS, an empty response, and
		// the tool it was refused. Without this it is a blank answer.
		p.Err = fmt.Errorf("agy stopped without answering: it needed permission to use %s, which sparstrowgen does not give agy yet",
			strings.Join(denied, ", "))
	}
	return p
}

type agyEvent struct {
	Event          string `json:"event"`
	ConversationID string `json:"conversation_id"`
	StepUpdate     struct {
		State     string `json:"state"`
		TextDelta string `json:"text_delta"`
	} `json:"step_update"`
	Result struct {
		ConversationID string   `json:"conversation_id"`
		Status         string   `json:"status"`
		Response       string   `json:"response"`
		Error          string   `json:"error"`
		Usage          agyUsage `json:"usage"`
		DeniedActions  []struct {
			Action      string `json:"action"`
			DisplayName string `json:"display_name"`
		} `json:"denied_actions"`
	} `json:"result"`
}

type agyUsage struct {
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	ThinkingTokens  int64 `json:"thinking_tokens"`
	CacheReadTokens int64 `json:"cache_read_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
}

func (u agyUsage) total() int64 {
	if u.TotalTokens > 0 {
		return u.TotalTokens
	}
	return u.InputTokens + u.OutputTokens + u.ThinkingTokens + u.CacheReadTokens
}
