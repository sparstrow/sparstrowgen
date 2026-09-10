# Deferred

Agreed to build, explicitly parked. Every entry records a concrete **trigger** for unparking, so
nothing sits here purely because it was forgotten. "Later" is not a trigger.

Distinct from [`Ideas.md`](Ideas.md): those were merely noticed, these have a decision behind them.

Written in the same turn the owner says "park it" / "later" / "not now", rather than relying on the
conversation being re-read. Ids are never reused.

---

## D-1 — Desktop app (Electron)

**Parked:** 2026-09-09, while locking the stack — the owner confirmed desktop and mobile come after
the web app.

The desktop app's real job is not a second UI; it is the daemon's installer, supervisor, and
updater. Until then the daemon runs as a terminal process, and later a Windows service. Nothing is
built, and `apps/desktop/` is an empty placeholder.

Cross-platform Electron packaging — notarization, per-arch builds, Linux binary naming, auto-update
wiring — is where the time goes, and none of it is worth paying while the owner is the only user.
A Go system-tray binary that shows daemon status and opens the web app should be evaluated before
Electron, since the daemon is already a Go binary and the web UI already exists.

- **If wrong:** nothing breaks. The daemon has to be started manually, which is friction for one
  person and unacceptable for a second user.
- **Unpark when:** someone other than the owner needs to run a daemon, or manually starting it
  becomes a daily annoyance.

---

## D-2 — Mobile app (Expo / React Native)

**Parked:** 2026-09-09, same conversation.

Not started; `apps/mobile/` is an empty placeholder. The web app is responsive and the preview
tunnel is designed to work in a phone browser, so a phone is reachable before any native app
exists.

- **If wrong:** phone use stays browser-only — no push notifications, no home-screen presence.
- **Unpark when:** the owner wants to be notified on his phone when an agent run finishes or needs
  a decision, which a web app cannot do well.

---

## D-3 — Permission approval UI

**Parked:** 2026-09-09, while deciding agent autonomy for the first feature.

Agents run with provider auto-approval flags, scoped to project directories registered with the
daemon; the daemon enforces the directory boundary rather than trusting the CLI. The approve/deny
round-trip surfaced in the chat window is the correct end state, and it is **additive** — a second
mode alongside auto, not a replacement for it. That is why building auto first is not throwaway
work.

- **If wrong:** an agent writes somewhere unexpected inside a registered directory. Bounded, and
  recoverable through git.
- **Unpark when:** an agent needs to run against a directory the owner does not fully trust, or a
  second person uses the app.
