package agent

import (
	"bytes"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

/* A turn's record: what was handed to the CLI and every line it printed back
   (docs/specs/2026-09-23-raw-exchange.md).

The parsers read a CLI's output for the few things the chat shows and drop the
rest. The record keeps the rest. It is taken beside the parsers, not by them:
stdout is read THROUGH a recorder on its way to the parser, and stderr, which no
parser reads, is written straight into one. So what a parser makes of a line and
what the record says was printed can never disagree about the bytes. */

// maxLine is how much of a single line the record keeps. The parser still sees
// every byte; only the copy is cut, and the cut is counted rather than hidden.
// Matches the daemon's per-turn budget, which a longer line could not fit anyway.
const maxLine = 16 << 20

// Line is one line a CLI printed, for the record.
type Line struct {
	Stream string
	// Milliseconds since the CLI was started.
	AtMs int64
	Text string
	// Cut is how many bytes of an over-long line were not kept.
	Cut int64
}

// splitter turns a byte stream into lines. Not safe for concurrent use; each
// stream has its own.
type splitter struct {
	stream string
	start  time.Time
	emit   func(Line)
	buf    []byte
	cut    int64
}

func (s *splitter) write(p []byte) {
	for len(p) > 0 {
		i := bytes.IndexByte(p, '\n')
		chunk := p
		if i >= 0 {
			chunk = p[:i]
		}
		room := maxLine - len(s.buf)
		if room >= len(chunk) {
			s.buf = append(s.buf, chunk...)
		} else {
			if room > 0 {
				s.buf = append(s.buf, chunk[:room]...)
				chunk = chunk[room:]
			}
			s.cut += int64(len(chunk))
		}
		if i < 0 {
			return
		}
		s.flush()
		p = p[i+1:]
	}
}

func (s *splitter) flush() {
	// The terminator is not part of the line; on Windows it is two bytes.
	text := string(bytes.TrimSuffix(s.buf, []byte("\r")))
	s.emit(Line{Stream: s.stream, AtMs: time.Since(s.start).Milliseconds(), Text: text, Cut: s.cut})
	s.buf = s.buf[:0]
	s.cut = 0
}

// end emits a last line that never got its newline, which is how a CLI that is
// killed mid-line leaves it.
func (s *splitter) end() {
	if len(s.buf) > 0 || s.cut > 0 {
		s.flush()
	}
}

// tapped passes a reader through unchanged, recording each line as it goes.
type tapped struct {
	r    io.Reader
	s    *splitter
	once sync.Once
}

func (t *tapped) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)
	if n > 0 {
		t.s.write(p[:n])
	}
	if err != nil {
		t.once.Do(t.s.end)
	}
	return n, err
}

// lineWriter is where os/exec copies stderr. It is written from os/exec's own
// goroutine, hence the lock.
type lineWriter struct {
	mu sync.Mutex
	s  *splitter
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.s.write(p)
	return len(p), nil
}

func (w *lineWriter) end() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.s.end()
}

// recording is one turn's record in the making.
type recording struct {
	start  time.Time
	out    chan<- Message
	stderr *lineWriter
}

// record attaches a recording to cmd before it starts: stderr goes to it, and
// stdout is read through stdout(). Lines are sent on out as MessageLine, in the
// order each stream produced them.
func record(cmd *exec.Cmd, out chan<- Message) *recording {
	r := &recording{start: time.Now(), out: out}
	r.stderr = &lineWriter{s: &splitter{stream: protocol.StreamStderr, start: r.start, emit: r.emit}}
	cmd.Stderr = r.stderr
	return r
}

func (r *recording) emit(l Line) {
	send(r.out, Message{Type: MessageLine, Line: &l})
}

func (r *recording) stdout(rd io.Reader) io.Reader {
	return &tapped{r: rd, s: &splitter{stream: protocol.StreamStdout, start: r.start, emit: r.emit}}
}

// finish flushes stderr's last line. Call it after the process has been
// reaped, because os/exec only stops writing to stderr once Wait returns.
func (r *recording) finish() {
	r.stderr.end()
}

// sentBy is what cmd was handed, for the record.
func sentBy(cmd *exec.Cmd, opts ExecOptions, prompt string, stdin []byte) protocol.ExchangeSent {
	return protocol.ExchangeSent{
		Program:         cmd.Path,
		Args:            append([]string{}, cmd.Args[1:]...),
		Cwd:             cmd.Dir,
		ResumeSessionID: opts.ResumeSessionID,
		Prompt:          prompt,
		Stdin:           string(stdin),
		Launched:        true,
	}
}
