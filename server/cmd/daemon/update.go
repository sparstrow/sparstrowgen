package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/release"
)

/* Keeping this copy updated (spec US3, docs/Decisions.md D-034, D-035).

A check reads the latest release's manifest from GitHub over HTTPS, and a newer
installer is downloaded and kept only if its SHA-256 matches the manifest's.
There is no signing key: trust is our GitHub releases, as Multica's. Installing never
interrupts agent work: it waits until no turn is running, then closes to new
turns under the same lock that counted them, so a turn cannot start in between.

The swap is done by a copy of THIS executable — the version known to work —
running as a separate updater process. It waits for this copy to exit, puts the
new executable in place and starts it, and puts this version back if the new
one has not reached the server within two minutes. */

// Set by the release build. A development build is "dev" with no update source,
// and never updates itself. releaseCandidateURL, the other stream, is in
// channel.go beside the code that decides which of the two this computer reads.
var (
	version          = "dev"
	releaseUpdateURL string
)

const (
	firstUpdateCheck = time.Minute
	updateCheckEvery = time.Hour
	updateTimeout    = 80 * time.Second
	healthWait       = 2 * time.Minute
	manifestLimit    = 64 << 10
	installerLimit   = 256 << 20
)

// Files in the updates directory, shared by the daemon and the updater process.
const (
	handoverFile  = "handover"    // the version the updater has just started
	connectedFile = "connected"   // the version that last reached the server
	resultFile    = "result.json" // why the last update was put back
	updaterExe    = "sparstrowgen-updater.exe"
)

func updatesDir() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "updates"), nil
}

type readyUpdate struct{ version, path string }

type updater struct {
	log     *slog.Logger
	turns   *runningTurns
	client  *http.Client
	source  string
	current string
	dir     string
	// start hands over to the updater process. Once it returns nil this copy
	// must exit, which is what exit does.
	start func(newExe, from, to string) error
	exit  func()
	send  func(protocol.DaemonMessage) error
	// poll is how often a waiting update looks at running work again.
	poll time.Duration

	mu        sync.Mutex
	automatic bool
	status    protocol.UpdateStatus
	ready     *readyUpdate
	// skip is a version whose install already failed. It is not retried by
	// itself; the person's "Try again" still installs it.
	skip       string
	activating bool
}

// newUpdater is nil when this copy cannot update itself: a development build, or
// one not running from its installed location.
func newUpdater(log *slog.Logger, turns *runningTurns, send func(protocol.DaemonMessage) error, exit func()) *updater {
	if !updatesSupported() {
		return nil
	}
	dir, err := updatesDir()
	if err != nil {
		log.Warn("updates are off: no data directory", "err", err)
		return nil
	}
	u := &updater{
		log: log, turns: turns, client: &http.Client{Timeout: updateTimeout},
		source: updateSource(), current: version, dir: dir,
		start: startUpdate, exit: exit, send: send, poll: 2 * time.Second,
		// On until the server says otherwise, which it does on every connect.
		automatic: true,
		status:    protocol.UpdateStatus{Kind: protocol.UpdateUnchecked},
	}
	u.resume()
	return u
}

// resume picks up what the updater process left behind.
func (u *updater) resume() {
	if r, ok := readUpdateResult(u.dir); ok {
		u.status = protocol.UpdateStatus{Kind: protocol.UpdateFailed, Message: r.Message}
		u.skip = r.Version
		_ = os.Remove(filepath.Join(u.dir, resultFile))
		return
	}
	if readText(filepath.Join(u.dir, handoverFile)) == u.current {
		u.status = protocol.UpdateStatus{Kind: protocol.UpdateCurrent}
	}
}

func (u *updater) Status() protocol.UpdateStatus {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.status
}

func (u *updater) setAutomatic(on bool) {
	u.mu.Lock()
	u.automatic = on
	u.mu.Unlock()
}

func (u *updater) setStatus(s protocol.UpdateStatus) {
	u.mu.Lock()
	u.status = s
	u.mu.Unlock()
	u.announce()
}

// announce tells the server where updates stand. Not being connected is fine:
// every connection announces again.
func (u *updater) announce() {
	s := u.Status()
	_ = u.send(protocol.DaemonMessage{Type: protocol.DaemonUpdateStatus, Update: &s})
}

// connected runs once the server has accepted this copy. It is the updater
// process's proof that a new version works.
func (u *updater) connected() {
	if err := writeText(filepath.Join(u.dir, connectedFile), u.current); err != nil {
		u.log.Warn("could not record the connection for the updater", "err", err)
	}
	u.announce()
}

func failedStatus(message string) protocol.UpdateStatus {
	return protocol.UpdateStatus{Kind: protocol.UpdateFailed, Message: message}
}

// check looks for a newer version and, if there is one, downloads and verifies
// it. It changes nothing on its own.
func (u *updater) check(ctx context.Context) (protocol.UpdateStatus, *readyUpdate) {
	still := fmt.Sprintf(" v%s is still running.", u.current)
	body, err := fetch(ctx, u.client, u.source, manifestLimit)
	if err != nil {
		return failedStatus(fmt.Sprintf("Could not check for updates (%v).", err)), nil
	}
	m, err := release.Parse(body, u.source)
	if err != nil {
		return failedStatus("The update information was not valid, so nothing was downloaded." + still), nil
	}
	later, err := release.Newer(m.Version, u.current)
	if err != nil {
		return failedStatus(fmt.Sprintf("Could not compare versions (%v).", err)), nil
	}
	if !later {
		return protocol.UpdateStatus{Kind: protocol.UpdateCurrent}, nil
	}
	path, err := download(ctx, u.client, m, u.dir)
	if errors.Is(err, release.ErrUntrusted) {
		return failedStatus(fmt.Sprintf("The download of v%s did not match its published checksum, so nothing was installed.%s", m.Version, still)), nil
	}
	if err != nil {
		return failedStatus(fmt.Sprintf("Could not download v%s (%v).%s", m.Version, err, still)), nil
	}
	return protocol.UpdateStatus{Kind: protocol.UpdateAvailable, Version: m.Version}, &readyUpdate{version: m.Version, path: path}
}

func (u *updater) busy() (protocol.UpdateStatus, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.status, u.activating
}

// CheckNow is the person pressing Check now. With automatic updates on, a newer
// version goes on to install once no agent work is running.
func (u *updater) CheckNow(ctx context.Context) protocol.UpdateStatus {
	if s, busy := u.busy(); busy {
		return s
	}
	s, ready := u.check(ctx)
	u.mu.Lock()
	u.ready = ready
	automatic := u.automatic
	u.mu.Unlock()
	if ready != nil && automatic {
		return u.beginActivation(ready, true)
	}
	u.setStatus(s)
	return s
}

// ApplyNow is Update now or Try again: install the newest version whatever the
// automatic setting, still waiting for running agent work.
func (u *updater) ApplyNow(ctx context.Context) protocol.UpdateStatus {
	if s, busy := u.busy(); busy {
		return s
	}
	s, ready := u.check(ctx)
	u.mu.Lock()
	u.ready = ready
	u.mu.Unlock()
	if ready == nil {
		u.setStatus(s)
		return s
	}
	return u.beginActivation(ready, true)
}

// automaticCheck is the hourly check. A version that already failed to install
// waits for the person to try again.
func (u *updater) automaticCheck(ctx context.Context) {
	u.mu.Lock()
	automatic, activating := u.automatic, u.activating
	u.mu.Unlock()
	if !automatic || activating {
		return
	}
	s, ready := u.check(ctx)
	u.mu.Lock()
	u.ready = ready
	skip := ready != nil && ready.version == u.skip
	u.mu.Unlock()
	switch {
	case ready == nil:
		u.setStatus(s)
	case skip:
	default:
		u.beginActivation(ready, false)
	}
}

func (u *updater) beginActivation(r *readyUpdate, deliberate bool) protocol.UpdateStatus {
	u.mu.Lock()
	if u.activating {
		s := u.status
		u.mu.Unlock()
		return s
	}
	u.activating = true
	u.mu.Unlock()
	s := protocol.UpdateStatus{Kind: protocol.UpdateUpdating, Version: r.version}
	if n := u.turns.count(); n > 0 {
		s = protocol.UpdateStatus{Kind: protocol.UpdateWaiting, Version: r.version, ActiveTasks: n}
	}
	u.setStatus(s)
	go u.activate(r, deliberate)
	return s
}

// activate waits for running agent work to end, however long it takes, and
// then hands over. Elapsed time is never permission to interrupt (spec US3).
func (u *updater) activate(r *readyUpdate, deliberate bool) {
	last := -1
	for {
		u.mu.Lock()
		automatic := u.automatic
		u.mu.Unlock()
		if !deliberate && !automatic {
			// Turned off while waiting: the current version stays until asked.
			u.mu.Lock()
			u.activating = false
			u.mu.Unlock()
			u.setStatus(protocol.UpdateStatus{Kind: protocol.UpdateAvailable, Version: r.version})
			return
		}
		if n := u.turns.count(); n > 0 {
			if n != last {
				u.setStatus(protocol.UpdateStatus{Kind: protocol.UpdateWaiting, Version: r.version, ActiveTasks: n})
				last = n
			}
			time.Sleep(u.poll)
			continue
		}
		// Checked again at the last moment, under the lock turns start under.
		if !u.turns.closeIfIdle() {
			continue
		}
		u.setStatus(protocol.UpdateStatus{Kind: protocol.UpdateUpdating, Version: r.version})
		if err := u.start(r.path, u.current, r.version); err != nil {
			u.turns.reopen()
			u.mu.Lock()
			u.activating = false
			u.mu.Unlock()
			u.setStatus(failedStatus(fmt.Sprintf("Could not start installing v%s (%v). v%s is still running.", r.version, err, u.current)))
			return
		}
		u.log.Info("handing over to the updater", "from", u.current, "to", r.version)
		u.exit()
		return
	}
}

// loop runs the automatic check shortly after starting and then every hour,
// while this copy is connected.
func (u *updater) loop(ctx context.Context, connected func() bool) {
	timer := time.NewTimer(firstUpdateCheck)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		if connected() {
			cctx, cancel := context.WithTimeout(ctx, updateTimeout)
			u.automaticCheck(cctx)
			cancel()
		}
		timer.Reset(updateCheckEvery)
	}
}

func fetch(ctx context.Context, client *http.Client, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the update server answered %s", res.Status)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, errors.New("the update server sent more than expected")
	}
	return body, nil
}

// download keeps the installer only if its SHA-256 matches the manifest.
// One already downloaded and matching is reused.
func download(ctx context.Context, client *http.Client, m release.Manifest, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "sparstrowgen-"+m.Version+".exe")
	if sum, err := release.FileSHA256(path); err == nil && sum == m.SHA256 {
		return path, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, nil)
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("the download server answered %s", res.Status)
	}
	partial := path + ".partial"
	f, err := os.OpenFile(partial, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(res.Body, installerLimit+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(partial)
		return "", err
	}
	if n > installerLimit || hex.EncodeToString(h.Sum(nil)) != m.SHA256 {
		_ = os.Remove(partial)
		return "", release.ErrUntrusted
	}
	if err := os.Rename(partial, path); err != nil {
		return "", err
	}
	removeOtherDownloads(dir, path)
	return path, nil
}

func removeOtherDownloads(dir, keep string) {
	matches, _ := filepath.Glob(filepath.Join(dir, "sparstrowgen-*.exe"))
	for _, m := range matches {
		if m != keep && filepath.Base(m) != updaterExe {
			_ = os.Remove(m)
		}
	}
}

type updateResult struct {
	Version string `json:"version"`
	Message string `json:"message"`
}

func readUpdateResult(dir string) (updateResult, bool) {
	raw, err := os.ReadFile(filepath.Join(dir, resultFile))
	if err != nil {
		return updateResult{}, false
	}
	var r updateResult
	if json.Unmarshal(raw, &r) != nil || r.Message == "" {
		return updateResult{}, false
	}
	return r, true
}

func writeUpdateResult(dir string, r updateResult) error {
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, resultFile), raw, 0o600)
}

func readText(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func writeText(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text+"\n"), 0o600)
}

// ---------------------------------------------------------------------------
// the updater process
// ---------------------------------------------------------------------------

type updateJob struct{ Target, New, From, To, Dir string }

// updateOps are the parts of an update that touch processes, injected so the
// sequence can be tested without replacing a real running executable.
type updateOps struct {
	// stopRunning waits until the running copy has exited.
	stopRunning func() error
	start       func(exe string) (exited <-chan struct{}, kill func(), err error)
	sleep       func(time.Duration)
	now         func() time.Time
}

// runUpdate puts the new version in place and keeps it only if it reaches the
// server within wait. On every failure a working copy is left running, and why
// is written down for that copy to report.
func runUpdate(log *slog.Logger, ops updateOps, job updateJob, wait time.Duration) error {
	giveUp := func(message string) error {
		// Recorded before starting anything: the copy started here reads it.
		if err := writeUpdateResult(job.Dir, updateResult{Version: job.To, Message: message}); err != nil {
			log.Error("could not record the failed update", "err", err)
		}
		if _, _, err := ops.start(job.Target); err != nil {
			log.Error("could not start sparstrowgen again", "err", err)
		}
		log.Error("update did not complete", "from", job.From, "to", job.To, "reason", message)
		return errors.New(message)
	}

	if err := ops.stopRunning(); err != nil {
		return giveUp(fmt.Sprintf("v%s was not installed because the running copy did not stop. v%s is still running.", job.To, job.From))
	}
	backup := fmt.Sprintf("%s.%d.old", job.Target, ops.now().UnixNano())
	if err := os.Rename(job.Target, backup); err != nil {
		return giveUp(fmt.Sprintf("v%s could not be installed (%v). v%s is still running.", job.To, err, job.From))
	}
	putBack := func() error {
		_ = os.Remove(job.Target)
		return os.Rename(backup, job.Target)
	}
	if err := copyFile(job.New, job.Target); err != nil {
		if perr := putBack(); perr != nil {
			log.Error("could not put the previous version back", "err", perr)
		}
		return giveUp(fmt.Sprintf("v%s could not be installed (%v). v%s is still running.", job.To, err, job.From))
	}

	_ = os.Remove(filepath.Join(job.Dir, connectedFile))
	if err := writeText(filepath.Join(job.Dir, handoverFile), job.To); err != nil {
		log.Warn("could not record the handover", "err", err)
	}
	defer os.Remove(filepath.Join(job.Dir, handoverFile))

	exited, kill, err := ops.start(job.Target)
	if err == nil {
		deadline := ops.now().Add(wait)
		for ops.now().Before(deadline) {
			if readText(filepath.Join(job.Dir, connectedFile)) == job.To {
				_ = os.Remove(backup)
				log.Info("updated", "from", job.From, "to", job.To)
				return nil
			}
			select {
			case <-exited:
				// It stopped by itself. Waiting longer cannot help.
				deadline = ops.now()
			default:
				ops.sleep(time.Second)
			}
		}
		kill()
		select {
		case <-exited:
		case <-time.After(10 * time.Second):
		}
	}

	// Windows releases a killed executable's file a moment after the process
	// ends, so putting the old one back is retried briefly.
	var perr error
	for i := 0; i < 20; i++ {
		if perr = putBack(); perr == nil {
			break
		}
		ops.sleep(500 * time.Millisecond)
	}
	if perr != nil {
		log.Error("could not put the previous version back", "err", perr)
	}
	return giveUp(fmt.Sprintf("v%s did not reconnect within %s, so v%s was put back and is running.", job.To, humanDuration(wait), job.From))
}

func humanDuration(d time.Duration) string {
	if d >= time.Minute && d%time.Minute == 0 {
		if n := int(d / time.Minute); n != 1 {
			return fmt.Sprintf("%d minutes", n)
		}
		return "1 minute"
	}
	return d.String()
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// ---------------------------------------------------------------------------
// the daemon's side
// ---------------------------------------------------------------------------

// answerUpdate answers a browser's Check now or Update now.
func (d *daemon) answerUpdate(ctx context.Context, msg protocol.ServerMessage) {
	var s protocol.UpdateStatus
	switch {
	case d.updates == nil:
		s = failedStatus("This copy of sparstrowgen cannot update itself. Download the installer and open it on this computer.")
	case msg.Type == protocol.ServerApplyUpdate:
		cctx, cancel := context.WithTimeout(ctx, updateTimeout)
		s = d.updates.ApplyNow(cctx)
		cancel()
	default:
		cctx, cancel := context.WithTimeout(ctx, updateTimeout)
		s = d.updates.CheckNow(cctx)
		cancel()
	}
	if err := d.send(protocol.DaemonMessage{Type: protocol.DaemonUpdateStatus, RequestID: msg.RequestID, Update: &s}); err != nil {
		d.log.Warn("update status not sent", "err", err)
	}
}

func (d *daemon) isConnected() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.conn != nil
}
