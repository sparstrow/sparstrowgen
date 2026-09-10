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

---

## D-4 — Process deliberately not carried over from the previous attempt

**Parked:** 2026-09-09, in two rounds — first while importing the process system from the earlier
sparstrowgen attempt, then again when the owner pointed out that *that process is why the previous
attempt never shipped*: the budget went into planning documents and conversation instead of code.

Two different reasons things were dropped, and the distinction matters when deciding whether to
bring one back.

### Obsoleted — capable coding agents removed the need

These existed because agents could not hold a plan in context and exercise judgment at the same
time. They should not come back in their old form.

| Dropped | What it did | Replaced by |
| --- | --- | --- |
| `decomposing-plans` skill | Split a plan into per-task files, a phase README, and a queue with concurrency tags | The plan itself carries the checklist, files, and verification |
| `docs/tasks/`, `MasterTaskQueue.md` | Task documents and global run order | Nothing. The plan is the last document before code |
| `task.md`, `verification-task.md`, `phase-spec.md` templates | Skeletons for the above | Nothing |
| `elaborating-ideas` skill | A procedure for writing an `Ideas.md` entry, with evidence gathering | A line or two, written directly |
| `slop-audit` skill + `slop-killer` agent | A separate report-only pass over a finished surface | Building with `ai-design-slop` loaded, then `frontend-verify` |
| `frontend-builder` agent | A subagent for building UI | The main session builds it |
| `register-entry.md` template | 219 lines of skeleton for four register formats | Each register states its own format at the top |
| `docs/bug/`, `docs/security/`, `docs/feedback/`, `docs/research/` | A directory, README, and template per category | One `Bugs.md`; feedback and research arrive in chat and become a spec, an idea, or a bug |

### Not obsolete — just wrong for *this* codebase, or too early

These need writing fresh, at the moment the thing they describe actually exists. Writing them
sooner means documenting an app that isn't built.

| Not imported | Why | Write it when |
| --- | --- | --- |
| `frontend-wiring` | Described a router mock, Zod contracts, and an in-app docs surface, none of which exist here | `apps/web` is scaffolded and has real wiring to describe |
| `designing-shared-contracts` | Was TypeScript-to-TypeScript via Zod; ours is Go-to-TypeScript via Protobuf | The first `proto/` message is written |
| `data-modeling-and-rls` | Built around Supabase row-level security; we are single-user on self-hosted Postgres | The first migration is written, as a sqlc/pgvector skill with no RLS |
| `release` | Vercel-specific | Deploying to Coolify |
| `worktree-orchestration` | Assumed an integration-branch tier and several agents in parallel | More than one agent runs on a feature at once |
| `migrate-radix-to-base` | A migration skill; this repo is greenfield shadcn | Never, most likely |
| `shadcn` | A global skill and MCP server already cover it | Never |
| `antigravity-guide`, `agy-customizations` | About using Antigravity as an IDE; we only drive its CLI | Never |
| `architect`, `scout`, `coordinator` agents | Orchestration roles; the lifecycle skills run fine in the main session for one person | Work is routinely handed to parallel agents |

The design chain (`design-brief` → `design-system` → `interactive-prototype`) was **kept but
gated**: it runs once, when real UI work starts, not per feature. A skill on disk costs nothing;
only invoking it spends tokens.

- **If wrong:** an agent improvises a procedure that used to be written down. Acceptable — a
  capable agent improvising beats a session spent writing documents nobody reads.
- **Unpark when:** the "write it when" condition in the second table is met. The originals are at
  `D:\My Setup\.claude\skills\` for reference.
