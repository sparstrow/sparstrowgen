package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/release"
)

/* The updater, against a real manifest served over TLS the way GitHub serves
it. Only the handover is faked: it would replace this test binary. */

type releaseServer struct {
	srv       *httptest.Server
	installer []byte
	manifest  atomic.Pointer[[]byte]
	// tamper serves different bytes from the ones the manifest names.
	tamper atomic.Bool
}

func newReleaseServer(t *testing.T, version string) *releaseServer {
	t.Helper()
	rs := &releaseServer{installer: []byte("sparstrowgen " + version)}
	mux := http.NewServeMux()
	rs.srv = httptest.NewTLSServer(mux)
	t.Cleanup(rs.srv.Close)
	rs.publish(t, version, rs.srv.URL+"/setup.exe")
	mux.HandleFunc("/update.json", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(*rs.manifest.Load()) })
	mux.HandleFunc("/setup.exe", func(w http.ResponseWriter, _ *http.Request) {
		if rs.tamper.Load() {
			_, _ = w.Write([]byte("something else entirely"))
			return
		}
		_, _ = w.Write(rs.installer)
	})
	return rs
}

// publish serves a manifest naming this server's installer at installerURL.
func (rs *releaseServer) publish(t *testing.T, version, installerURL string) {
	t.Helper()
	sum := sha256.Sum256(rs.installer)
	body, err := release.Encode(release.Manifest{Version: version, URL: installerURL, SHA256: hex.EncodeToString(sum[:])})
	if err != nil {
		t.Fatal(err)
	}
	rs.manifest.Store(&body)
}

type statusLog struct {
	mu   sync.Mutex
	seen []protocol.UpdateStatus
}

func (s *statusLog) send(m protocol.DaemonMessage) error {
	if m.Update != nil {
		s.mu.Lock()
		s.seen = append(s.seen, *m.Update)
		s.mu.Unlock()
	}
	return nil
}

type handover struct{ newExe, from, to string }

type updaterRig struct {
	u       *updater
	sent    *statusLog
	started chan handover
	exited  chan struct{}
}

func newUpdaterRig(t *testing.T, rs *releaseServer, current string) *updaterRig {
	t.Helper()
	r := &updaterRig{sent: &statusLog{}, started: make(chan handover, 4), exited: make(chan struct{}, 4)}
	r.u = &updater{
		log: quietLog(), turns: newRunningTurns(), client: rs.srv.Client(),
		source: rs.srv.URL + "/update.json", current: current, dir: t.TempDir(),
		start: func(newExe, from, to string) error { r.started <- handover{newExe, from, to}; return nil },
		exit:  func() { r.exited <- struct{}{} },
		send:  r.sent.send, poll: 10 * time.Millisecond, automatic: true,
		status: protocol.UpdateStatus{Kind: protocol.UpdateUnchecked},
	}
	return r
}

func (r *updaterRig) handedOver(t *testing.T) handover {
	t.Helper()
	select {
	case h := <-r.started:
		select {
		case <-r.exited:
		case <-time.After(5 * time.Second):
			t.Fatal("handed over but did not exit")
		}
		return h
	case <-time.After(5 * time.Second):
		t.Fatal("never handed over to the updater")
		return handover{}
	}
}

func (r *updaterRig) neverHandedOver(t *testing.T, d time.Duration) {
	t.Helper()
	select {
	case h := <-r.started:
		t.Fatalf("handed over when it must not have: %+v", h)
	case <-time.After(d):
	}
}

func (r *updaterRig) awaitStatus(t *testing.T, want func(protocol.UpdateStatus) bool) protocol.UpdateStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if s := r.u.Status(); want(s) {
			return s
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("status never arrived; last %+v", r.u.Status())
	return protocol.UpdateStatus{}
}

func TestACheckWithNothingNewerSaysCurrent(t *testing.T) {
	rs := newReleaseServer(t, "0.2.0")
	r := newUpdaterRig(t, rs, "0.2.0")
	if s := r.u.CheckNow(context.Background()); s.Kind != protocol.UpdateCurrent {
		t.Fatalf("status = %+v, want current", s)
	}
	r.neverHandedOver(t, 100*time.Millisecond)
}

func TestAVerifiedUpdateInstallsWhenNothingIsRunning(t *testing.T) {
	rs := newReleaseServer(t, "0.2.1")
	r := newUpdaterRig(t, rs, "0.2.0")
	if s := r.u.CheckNow(context.Background()); s.Kind != protocol.UpdateUpdating || s.Version != "0.2.1" {
		t.Fatalf("status = %+v, want updating to 0.2.1", s)
	}
	h := r.handedOver(t)
	if h.from != "0.2.0" || h.to != "0.2.1" {
		t.Errorf("handover = %+v", h)
	}
	if got, err := os.ReadFile(h.newExe); err != nil || string(got) != string(rs.installer) {
		t.Errorf("the handed-over installer is not the verified download: %q, %v", got, err)
	}
	if r.u.turns.begin("late", func() {}) || !r.u.turns.closedForUpdate() {
		t.Error("a turn could start after handing over")
	}
}

// Spec US3: an update never stops, replaces or restarts a copy with agent work
// running, and waits however long that work takes.
func TestAnUpdateWaitsUntilEveryRunningTaskHasEnded(t *testing.T) {
	rs := newReleaseServer(t, "0.2.1")
	r := newUpdaterRig(t, rs, "0.2.0")
	r.u.turns.begin("a", func() {})
	r.u.turns.begin("b", func() {})

	if s := r.u.CheckNow(context.Background()); s.Kind != protocol.UpdateWaiting || s.ActiveTasks != 2 {
		t.Fatalf("status = %+v, want waiting on 2 tasks", s)
	}
	r.neverHandedOver(t, 150*time.Millisecond)

	r.u.turns.end("a")
	r.awaitStatus(t, func(s protocol.UpdateStatus) bool { return s.Kind == protocol.UpdateWaiting && s.ActiveTasks == 1 })
	r.neverHandedOver(t, 100*time.Millisecond)

	r.u.turns.end("b")
	r.handedOver(t)
}

// The last-moment check: closing to new turns only succeeds with none running,
// under the same lock a turn starts under.
func TestClosingForAnUpdateFailsWhileATurnIsRunning(t *testing.T) {
	turns := newRunningTurns()
	turns.begin("a", func() {})
	if turns.closeIfIdle() {
		t.Fatal("closed with a turn running")
	}
	if !turns.begin("b", func() {}) {
		t.Fatal("a failed close refused a new turn")
	}
	turns.end("a")
	turns.end("b")
	if !turns.closeIfIdle() {
		t.Fatal("did not close when idle")
	}
	if turns.begin("c", func() {}) || !turns.closedForUpdate() {
		t.Fatal("a turn started after closing")
	}
	turns.reopen()
	if !turns.begin("d", func() {}) {
		t.Fatal("reopening did not accept turns again")
	}
}

func TestWithAutomaticUpdatesOffACheckOnlyReportsTheUpdate(t *testing.T) {
	rs := newReleaseServer(t, "0.2.1")
	r := newUpdaterRig(t, rs, "0.2.0")
	r.u.setAutomatic(false)
	if s := r.u.CheckNow(context.Background()); s.Kind != protocol.UpdateAvailable || s.Version != "0.2.1" {
		t.Fatalf("status = %+v, want available", s)
	}
	r.u.automaticCheck(context.Background())
	r.neverHandedOver(t, 150*time.Millisecond)

	// Update now is deliberate, so it installs whatever the setting.
	if s := r.u.ApplyNow(context.Background()); s.Kind != protocol.UpdateUpdating {
		t.Fatalf("Update now = %+v, want updating", s)
	}
	r.handedOver(t)
}

func TestADownloadThatDoesNotMatchItsChecksumIsRefused(t *testing.T) {
	rs := newReleaseServer(t, "0.2.1")
	rs.tamper.Store(true)
	r := newUpdaterRig(t, rs, "0.2.0")
	s := r.u.CheckNow(context.Background())
	if s.Kind != protocol.UpdateFailed || !strings.Contains(s.Message, "did not match its published checksum") || !strings.Contains(s.Message, "v0.2.0 is still running") {
		t.Fatalf("status = %+v", s)
	}
	if matches, _ := filepath.Glob(filepath.Join(r.u.dir, "sparstrowgen-*")); len(matches) != 0 {
		t.Errorf("kept an unverified download: %v", matches)
	}
	r.neverHandedOver(t, 100*time.Millisecond)
}

// A manifest can never send a computer to download from another site, even with
// that installer's correct checksum.
func TestAManifestNamingAnInstallerElsewhereIsRefused(t *testing.T) {
	rs := newReleaseServer(t, "0.2.1")
	elsewhere := newReleaseServer(t, "0.2.1")
	rs.publish(t, "0.2.1", elsewhere.srv.URL+"/setup.exe")
	r := newUpdaterRig(t, rs, "0.2.0")
	s := r.u.CheckNow(context.Background())
	if s.Kind != protocol.UpdateFailed || !strings.Contains(s.Message, "was not valid") || !strings.Contains(s.Message, "v0.2.0 is still running") {
		t.Fatalf("status = %+v", s)
	}
	if matches, _ := filepath.Glob(filepath.Join(r.u.dir, "sparstrowgen-*")); len(matches) != 0 {
		t.Errorf("downloaded from elsewhere: %v", matches)
	}
	r.neverHandedOver(t, 100*time.Millisecond)
}

func TestAVersionThatFailedIsNotRetriedByItself(t *testing.T) {
	rs := newReleaseServer(t, "0.2.1")
	r := newUpdaterRig(t, rs, "0.2.0")
	r.u.skip = "0.2.1"
	r.u.automaticCheck(context.Background())
	r.neverHandedOver(t, 150*time.Millisecond)

	r.u.ApplyNow(context.Background())
	r.handedOver(t)
}

func TestTurningAutomaticUpdatesOffStopsAnAutomaticWait(t *testing.T) {
	rs := newReleaseServer(t, "0.2.1")
	r := newUpdaterRig(t, rs, "0.2.0")
	r.u.turns.begin("a", func() {})
	r.u.automaticCheck(context.Background())
	r.awaitStatus(t, func(s protocol.UpdateStatus) bool { return s.Kind == protocol.UpdateWaiting })

	r.u.setAutomatic(false)
	r.awaitStatus(t, func(s protocol.UpdateStatus) bool { return s.Kind == protocol.UpdateAvailable })
	r.u.turns.end("a")
	r.neverHandedOver(t, 150*time.Millisecond)
}

func TestTheCopyPutBackAfterAFailedUpdateSaysWhy(t *testing.T) {
	dir := t.TempDir()
	if err := writeUpdateResult(dir, updateResult{Version: "0.2.1", Message: "v0.2.1 did not reconnect within 2 minutes, so v0.2.0 was put back and is running."}); err != nil {
		t.Fatal(err)
	}
	u := &updater{dir: dir, current: "0.2.0", status: protocol.UpdateStatus{Kind: protocol.UpdateUnchecked}}
	u.resume()
	if u.status.Kind != protocol.UpdateFailed || !strings.Contains(u.status.Message, "put back") || u.skip != "0.2.1" {
		t.Errorf("after resume: status %+v, skip %q", u.status, u.skip)
	}
	if _, ok := readUpdateResult(dir); ok {
		t.Error("the result was reported and kept, so it would be reported again")
	}

	// The new copy the updater started says it is current.
	if err := writeText(filepath.Join(dir, handoverFile), "0.2.1"); err != nil {
		t.Fatal(err)
	}
	fresh := &updater{dir: dir, current: "0.2.1", status: protocol.UpdateStatus{Kind: protocol.UpdateUnchecked}}
	fresh.resume()
	if fresh.status.Kind != protocol.UpdateCurrent {
		t.Errorf("a freshly updated copy says %+v", fresh.status)
	}
}

// ---------------------------------------------------------------------------
// the updater process
// ---------------------------------------------------------------------------

type fakeProcesses struct {
	mu      sync.Mutex
	started []string
	clock   time.Time
	// behave decides, for the nth start, whether the process reaches the
	// server and whether it stops by itself.
	behave func(n int) (connects, crashes bool)
	dir    string
	to     string
}

func (f *fakeProcesses) ops(stopErr error) updateOps {
	return updateOps{
		stopRunning: func() error { return stopErr },
		start: func(exe string) (<-chan struct{}, func(), error) {
			f.mu.Lock()
			f.started = append(f.started, exe)
			n := len(f.started)
			f.mu.Unlock()
			exited := make(chan struct{})
			var once sync.Once
			kill := func() { once.Do(func() { close(exited) }) }
			connects, crashes := f.behave(n)
			if connects {
				_ = writeText(filepath.Join(f.dir, connectedFile), f.to)
			}
			if crashes {
				kill()
			}
			return exited, kill, nil
		},
		sleep: func(d time.Duration) { f.mu.Lock(); f.clock = f.clock.Add(d); f.mu.Unlock() },
		now:   func() time.Time { f.mu.Lock(); defer f.mu.Unlock(); return f.clock },
	}
}

func updateFiles(t *testing.T) (updateJob, *fakeProcesses) {
	t.Helper()
	root := t.TempDir()
	job := updateJob{Target: filepath.Join(root, "sparstrowgen.exe"), New: filepath.Join(root, "new.exe"), From: "0.2.0", To: "0.2.1", Dir: filepath.Join(root, "updates")}
	if err := os.WriteFile(job.Target, []byte("old version"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(job.New, []byte("new version"), 0o755); err != nil {
		t.Fatal(err)
	}
	return job, &fakeProcesses{clock: time.Unix(1_800_000_000, 0), dir: job.Dir, to: job.To}
}

func contents(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestAnUpdateThatReconnectsIsKept(t *testing.T) {
	job, procs := updateFiles(t)
	procs.behave = func(int) (bool, bool) { return true, false }
	if err := runUpdate(quietLog(), procs.ops(nil), job, time.Minute); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if got := contents(t, job.Target); got != "new version" {
		t.Errorf("installed = %q", got)
	}
	if backups, _ := filepath.Glob(job.Target + ".*.old"); len(backups) != 0 {
		t.Errorf("left the previous version behind: %v", backups)
	}
	if _, ok := readUpdateResult(job.Dir); ok {
		t.Error("recorded a failure for an update that worked")
	}
}

// Spec US3 and the phase 1 exit gate: an update that cannot reconnect is rolled
// back, and the copy put back can say why.
func TestAnUpdateThatNeverReconnectsIsPutBack(t *testing.T) {
	job, procs := updateFiles(t)
	procs.behave = func(int) (bool, bool) { return false, false }
	err := runUpdate(quietLog(), procs.ops(nil), job, 2*time.Minute)
	if err == nil {
		t.Fatal("reported success")
	}
	if got := contents(t, job.Target); got != "old version" {
		t.Errorf("after rollback the installed copy is %q", got)
	}
	r, ok := readUpdateResult(job.Dir)
	if !ok || r.Version != "0.2.1" || !strings.Contains(r.Message, "did not reconnect within 2 minutes, so v0.2.0 was put back") {
		t.Errorf("result = %+v, %v", r, ok)
	}
	if len(procs.started) != 2 || procs.started[1] != job.Target {
		t.Errorf("starts = %v, want the new copy and then the old one", procs.started)
	}
}

func TestAnUpdateThatStopsAtOnceIsPutBackWithoutWaiting(t *testing.T) {
	job, procs := updateFiles(t)
	began := procs.clock
	procs.behave = func(n int) (bool, bool) { return false, n == 1 }
	if err := runUpdate(quietLog(), procs.ops(nil), job, 2*time.Minute); err == nil {
		t.Fatal("reported success")
	}
	if got := contents(t, job.Target); got != "old version" {
		t.Errorf("installed = %q", got)
	}
	if waited := procs.clock.Sub(began); waited >= 2*time.Minute {
		t.Errorf("waited %s for a copy that had already stopped", waited)
	}
}

func TestWhenTheRunningCopyWillNotStopNothingIsReplaced(t *testing.T) {
	job, procs := updateFiles(t)
	procs.behave = func(int) (bool, bool) { return false, false }
	if err := runUpdate(quietLog(), procs.ops(os.ErrDeadlineExceeded), job, time.Minute); err == nil {
		t.Fatal("reported success")
	}
	if got := contents(t, job.Target); got != "old version" {
		t.Errorf("installed = %q", got)
	}
	if r, ok := readUpdateResult(job.Dir); !ok || !strings.Contains(r.Message, "did not stop") {
		t.Errorf("result = %+v, %v", r, ok)
	}
}

// ---------------------------------------------------------------------------
// hello
// ---------------------------------------------------------------------------

func TestHelloSaysWhichVersionAndProtocolThisIs(t *testing.T) {
	hello := make(chan protocol.DaemonMessage, 1)
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		var msg protocol.DaemonMessage
		if err := conn.ReadJSON(&msg); err == nil {
			hello <- msg
		}
	}))
	defer srv.Close()

	d := &daemon{log: quietLog(), turns: newRunningTurns(),
		detect: func(context.Context) []protocol.Provider { return nil }}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.run(ctx, "ws"+strings.TrimPrefix(srv.URL, "http"), "credential") }()

	select {
	case msg := <-hello:
		if msg.Type != protocol.DaemonHello || msg.Version != version || msg.Protocol != protocol.DaemonProtocol || msg.SelfUpdates {
			t.Errorf("hello = type %q version %q protocol %d selfUpdates %v", msg.Type, msg.Version, msg.Protocol, msg.SelfUpdates)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no hello")
	}
}
