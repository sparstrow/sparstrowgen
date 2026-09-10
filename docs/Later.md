# Later

Everything not being built right now, in one place, so parking something is never a decision about
which file it goes in.

```
## L-n — <the thing>
**Status:** question | parked | idea      **Raised:** <YYYY-MM-DD>
<Two or three lines. What it is, and why it isn't happening now.>
**Unblocks when:** <a concrete trigger — an event, a threshold, a person. Not "later".>
```

| Status | Meaning |
|---|---|
| `question` | Needs a decision from the owner. Blocks only the one thing that depends on it — build everything else and report "done except L-n" |
| `parked` | Agreed in principle, deliberately not now |
| `idea` | Noticed, no commitment, may never happen |

Write it in the same turn the owner says "park it" / "later" / "just an idea". Delete an entry when
it's done or dead — the length of this file is a real signal. Ids are never reused.

**A `question` should be rare.** The standing preference is that agents recommend and the owner
vetoes. And if a choice can be *rendered*, show him designs instead of writing it down here.

---

## L-1 — Remote desktop access to the machine

**Status:** idea **Raised:** 2026-09-09

Seeing and controlling the machine from a phone. **Not built here** — commodity, multi-year work;
deploy MeshCentral or Guacamole and embed the viewer behind our own auth. Reasoning in
[`Decisions.md`](Decisions.md) D-006.

Do not confuse it with the **preview tunnel** (seeing the app the agent just built on localhost),
which is a planned feature and strictly better for that job — crisp text instead of video, real
touch interaction, no encoding latency.

**Unblocks when:** something needs reaching a *native* app on the machine, not a web app.

## L-2 — Desktop app (Electron)

**Status:** parked **Raised:** 2026-09-09

`apps/desktop/` is an empty placeholder. The desktop app's real job is being the daemon's
installer, supervisor, and updater — not a second UI. Until then the daemon runs as a terminal
process, then a Windows service. A Go system-tray binary should be evaluated before Electron,
since the daemon is already a Go binary and the web UI already exists.

**Unblocks when:** someone other than the owner needs to run a daemon, or starting it manually
becomes a daily annoyance.

## L-3 — Mobile app (Expo)

**Status:** parked **Raised:** 2026-09-09

`apps/mobile/` is an empty placeholder. The web app is responsive and the preview tunnel is
designed for a phone browser, so a phone is reachable before any native app exists.

**Unblocks when:** the owner wants push notification when a run finishes or needs a decision —
which a web app can't do well.

## L-4 — Permission approval UI

**Status:** parked **Raised:** 2026-09-09

Agents run auto-approved inside directories registered with the daemon, which enforces the
boundary rather than trusting the CLI. The approve/deny round-trip is **additive** — a second mode
alongside auto, not a replacement — which is why building auto first isn't throwaway work. See
[`Decisions.md`](Decisions.md) D-007.

**Unblocks when:** an agent must run against a directory the owner doesn't fully trust, or a second
person uses the app.

## L-5 — Skills to write when the thing they describe exists

**Status:** parked **Raised:** 2026-09-09

Carried over from the previous attempt but not imported, because their mechanics described that
codebase. Originals at `D:\My Setup\.claude\skills\` for reference.

| Skill | Write it when |
|---|---|
| Frontend wiring | `apps/web` is scaffolded and has real wiring to describe |
| Go↔TS contract | The first `proto/` message is written |
| Data modelling | The first migration is written — sqlc and pgvector, no row-level security |
| Release | Deploying to Coolify |
| Worktree orchestration | More than one agent runs on a feature at once |

Multica's multi-worktree environment registry is the best version of that last one and worth
copying when it's needed — see [`Decisions.md`](Decisions.md) D-009.

**Unblocks when:** the row's condition is met. Not before — writing them earlier means documenting
an app that isn't built.
