# Decisions

Load-bearing technical choices, with what was rejected and why. Append-only; a few lines each, not
a document. The code shows what was built — this is the only place that shows why the alternatives
lost.

Add an entry when a choice would be expensive to reverse: schema, protocol, transport, a boundary
between components. Skip it for choices that are cheap to change later.

`.sparstrowgen/blueprint.yaml` holds *what* the stack is. This holds *why*.

---

## D-001 — Go for both the server and the daemon

**2026-09-09.** Rejected: TypeScript everywhere, Rust everywhere, Go server with a Rust daemon.

The daemon is a genuinely native program — process groups, signals, PTYs, file watching — and
cross-compiles to a single dependency-free binary, which Node cannot match without native addons.
Rust would win on raw capability but its async complexity buys nothing for what is ~90% I/O
orchestration, and compile times matter when iterating with agents.

The one thing that argued for Rust was screen capture and video encoding. That disappeared when
"screen sharing" turned out to mean a reverse HTTP tunnel to a localhost dev server, not pixels.

## D-002 — Server and daemon share one Go module

**2026-09-09.** Rejected: separate modules with a shared protocol package.

Both binaries build from `server/`. The wire protocol structs are used by both, so drift becomes a
compile error rather than a runtime bug on a laptop running an older daemon.

## D-003 — Protobuf for the wire protocol

**2026-09-09.** Rejected: hand-maintained JSON types on both sides.

One `.proto` generates Go structs and TypeScript types. The alternative is the fast option that
gets replaced: the moment an installed daemon lags the server, untyped JSON becomes a runtime bug
instead of a compile error. The codegen step is paid once, in week one.

## D-004 — Our database is the transcript of record; provider sessions are a cache

**2026-09-09.** Rejected: treating a provider's own session as the source of truth.

No agent CLI can resume another's session, so cross-provider switching — the product's whole
reason to exist — is impossible unless we own the conversation in a provider-neutral format. A
`provider_session` row per (conversation, provider) with a last-seen sequence number means
switching back later replays only the gap, not the whole history.

The schema is multi-provider from the first migration for the same reason. Retrofitting it is
exactly the rewrite this project is trying to avoid.

## D-005 — Embeddings run behind HTTP, never in-process

**2026-09-09.** Rejected: an in-process embedding library; a Python service for the whole backend.

pgvector lives in the Postgres we already run; embeddings come from a separate container (Ollama
or text-embeddings-inference). This keeps the backend language decision free of the ML ecosystem
— vector search is SQL, and embedding is an HTTP call. Python only earns a place if we start
training models, and that arrives as its own service behind the same boundary.

## D-006 — Remote desktop is deployed, not built

**2026-09-09.** Rejected: building screen capture, encoding, and input injection.

Commodity, multi-engineer-year work. Deploy MeshCentral or Guacamole and embed the viewer behind
our own auth. Full reasoning in [`Later.md`](Later.md) L-1.

## D-007 — Agents run auto-approved inside registered directories

**2026-09-09.** Rejected: plan/read-only first; a full permission UI first.

Auto-approval scoped to directories registered with the daemon is what actually replaces the
owner's desktop apps, and the daemon enforces the boundary rather than trusting the CLI. The
approve/deny UI is **additive** — a second mode alongside auto — which is why building auto first
is not throwaway work. Parked as [`Later.md`](Later.md) L-4.

## D-008 — Design-driven development; a spec, then designs, no plan

**2026-09-09.** Rejected: spec → plan → tasks → code, which is what the previous attempt did.

Features start with a spec the owner approves, then two or three rendered directions he picks
from. The chosen one is wired into the real app on placeholder data, and the backend is built only
once he has confirmed it there. The guardrail is [`Capabilities.md`](Capabilities.md): design
nothing the backend cannot deliver.

**The plan layer is gone, not the spec.** Cutting both was briefly tried and was wrong: the spec is
the one document the *owner* authors, where his scenarios get elaborated, and a design cannot
replace it — a design shows what a thing looks like, not why it should exist or what should be
true afterwards. What a plan carried moved into the prototype's handoff contract, which is derived
from an approved design and so cannot describe a feature nobody asked for.

The two must not overlap: a spec that describes an interface pre-empts the options the owner is
supposed to choose between, and quietly becomes the design decision.

## D-009 — What we took from Multica, and what we overrode

**2026-09-09.** [Multica](../Reference/multica-main) is a live, working system of the same shape —
Go server, local daemon driving agent CLIs, Next.js front end. Its conventions were reviewed
wholesale rather than reinvented.

**Adopted more or less as-is**, because it hit these problems at scale first:

- **Testing discipline** — where tests live, one canonical layer per behaviour, `node` environment
  for DOM-free tests, shared Go fixtures instead of open-coded inserts, and helpers that never
  assert a product rule on a test's behalf. The `testing` skill.
- **Never letting a default test execute a real agent CLI.** The single most valuable rule they
  have for us: we drive the same CLIs, and a test that resolves one from `PATH` spends the owner's
  quota. Fake executable paths by default; real-agent smoke behind a build tag *and* an env var.
- **Server/client state separation** — TanStack Query owns server state, Zustand owns view state,
  realtime events never mirror payloads into Zustand. Same stack, same bug class.
- **Package boundaries** as hard constraints, so the desktop shell stays possible later.
- **One command for the full pipeline** (`make check`: typecheck → unit → Go → e2e).
- **Compatibility discipline at the boundary** — enum switches need a default branch, don't pin an
  affordance to one boolean. Their concern is old desktop clients; ours is an old daemon.

**Overrode, deliberately:**

- **Foreign keys.** Multica bans them outright and resolves every relationship in application
  code. That earns its place at their scale and with polymorphic assignees; here it would trade a
  free correctness guarantee for hand-written integrity checks. **We keep foreign keys and ban
  cascading deletes** — cascades are the actual hazard they were guarding against, and a surprise
  recursive delete is much worse than a rejected insert.
- **`CREATE INDEX CONCURRENTLY` on every index, one statement per migration file.** Correct
  against a live table under load; premature against zero rows, and it forces the migration runner
  outside a transaction. Revisit when there is production data — until then, ordinary indexes.

**Not taken:** their multi-worktree development-environment registry (`make up`, per-checkout port
and database allocation), reserved slugs, i18n glossary, and the mobile and desktop rule sets.
Each solves a problem we do not have yet — several agents on one machine, several locales,
several shells. The registry in particular is excellent and worth revisiting the day a second
agent runs here.

## D-010 — Caveats fold into KnownGaps, rather than getting their own file

**2026-09-09.** Rejected: a separate `Caveats.md`.

Something an agent notices in passing and deliberately leaves alone answers the same question as
an unproved claim: *how much can the next agent take on faith?* `KnownGaps.md` is already the file
read before relying on an area, so a second register would be another place to look and another
place to forget. Entries carry a `Kind:` of `unproved` or `caveat` so both stay legible.

## D-011 — Provider availability is three-valued: available / waitable / blocked

**2026-09-09.** Rejected: a plain online/offline boolean.

Adopted from Multica's `AgentAvailability` (`server/internal/service/agent_ready.go`): the
distinction that matters is not "ready or not" but *whether waiting is a plan*. A daemon whose
machine is asleep resolves itself — queue the work. A CLI that is installed but cannot run (no
account, no execute permission, broken postinstall) will never resolve until a human acts, and a
caller must refuse and say why rather than queue silently.

This surfaced immediately: `gemini` was installed on the owner's machine with no account signed in.
It was **blocked**, not "not built" — a real, permanent-until-fixed state a provider picker must
show with a reason, the way Multica's `RuntimeUnusableNotice` does, not silently omit the way an
unbuilt provider is omitted.

**That example is gone** — gemini was removed from the product on 2026-09-10 (D-014), so `blocked`
now has no live case. The decision stands: an uninstalled or signed-out CLI is the same shape, and
the daemon still has to tell it apart from "temporarily out". It is simply an unexercised path now,
and the first real one should be checked against the design rather than assumed to fit.

## D-012 — An image round sits between the spec and the prototype

**2026-09-09.** Rejected: going straight from spec to clickable prototypes, which is what the
workflow did until now.

Coding a prototype is the right way to *decide* a design but the wrong way to *explore* one. It
costs enough that exploration silently caps at two directions, and the second is usually a
variation on the first — so the direction that gets built is the first one anybody thought of, not
one that was chosen. `codex` generates a usable UI mockup in about two minutes (verified
2026-09-09, ~88k input tokens, mostly cached preamble), which makes four genuinely different
directions affordable before a line of code exists.

The ordering principle generalises what the workflow already did with the backend: **each step is
cheaper than the one after it, so the expensive step only ever runs on something already wanted.**
Shots are to prototypes what prototypes are to the backend.

Two constraints keep it from becoming waste rather than removing it:

- **An image decides a direction, never the design.** It has no interaction, no real data, no four
  states. The prototype step still makes the decision, and the owner is told that when the images
  are presented — otherwise the prototype feels like re-opening a settled question.
- **The feasibility gate applies before the first prompt.** A rendered image of an undeliverable
  screen is *more* dangerous than a sketch of one, because polish buys approval.

Adapted from [garrytan/gstack's `design-shotgun`](https://github.com/garrytan/gstack/tree/main/design-shotgun):
took the anti-convergence rule, confirming concepts before spending, and the side-by-side board.
Left its external artifact tree, JSON handshake files, polling loops, and a taste profile with
weekly-decaying confidence scores — that is the planning machinery that killed the first attempt.
Taste is recorded instead as one `README.md` per shot round, holding the owner's stated reasons; a
preference seen twice gets promoted into `DESIGN.md`.

## D-013 — The prototype is built in the real app, not as a standalone artifact

**2026-09-10.** Rejected: a self-contained `.dc.html` prototype in `design-system/designs/`, which
is what the `interactive-prototype` skill prescribes and what was half-built before the owner
stopped it.

He asked, mid-build, whether shadcn was actually being used. It was not — the prototype was plain
HTML with hand-written CSS, because that is what opens from disk with no build step. That
portability is the skill's whole argument for the format, and here it buys nothing: the real app is
Next.js + shadcn and must exist regardless, so a hand-rolled prototype is not a cheaper rehearsal,
it is the same screen built twice in two different technologies. The second build would also
silently re-decide spacing, component behaviour and states, because none of it transfers.

So the Shots → Design → Wire sequence collapses its last two steps **when the app does not exist
yet**: the chosen direction is built directly in `apps/web` on `*.mock.ts` data. The ordering
principle is untouched — cheap steps still gate expensive ones, and images still cost two minutes
against a day. What changed is the recognition that once a real app exists, wiring into it *is* the
cheapest way to see a design, because it is the only build that survives.

This does not retire `interactive-prototype`. It stays right for a surface whose route does not
belong in the app yet, or for comparing two layouts side by side without committing either.

**What the design's own contract now demands of the backend**, falling out of the built surface:

- A conversation is provider-neutral and carries `seenBy` per provider, so a switch back replays
  only the gap. This is the transcript-of-record decision (D-004) becoming a concrete field.
- **Replay is lazy.** Selecting a provider is free and changes nothing server-side; the catch-up is
  performed on the first message sent afterwards, and the cost is written into the transcript at
  that point. The owner caught this — switching casually must not burn tokens — and it is a
  data-model constraint, not a UI nicety: a `pending_switch` is client state and never persists.
- Progress must be reportable independently of output text, because `codex` emits no deltas at all.
- Usage is asymmetric by provider and the UI must not imply otherwise: USD for `claude`, tokens
  only for `codex` and `agy`, and "no limit data" is a distinct state from "plenty left".

## D-014 — `gemini` is removed from the product, not parked

**2026-09-10.** Rejected: keeping it as a permanently `blocked` provider, which is what the surface
shipped with and what `Later.md` L-6 planned for.

The owner ruled it out directly — *"I don't want gemini to be part of this at all, since we are not
going to use it"* — so it comes out of `ProviderId`, the colour tokens, the mock and the blueprint,
and L-6 is deleted rather than left parked. A provider nobody will sign into is not a future
feature; it is a row that makes every list longer and every switch statement wider for nothing.

**The cost, stated plainly:** `gemini` was the only live example of the three-valued availability
model (D-011). `blocked` now has no case behind it in the running app. D-011 is *not* reversed —
the daemon still has to distinguish "installed but unusable" from "temporarily out", and a CLI that
is uninstalled or signed out is the same shape. But it is now an unexercised path, and the first
real one should be checked against the design rather than assumed to still fit.

## D-015 — A model is an id and a label, and a transcript stores both

**2026-09-10.** Rejected: `model: string`, which is what the surface shipped with.

`agy models` settled it by printing two columns — `gemini-3.1-pro-high` alongside
`Gemini 3.1 Pro (High)`. The id is what gets passed to the CLI; the label is the only thing worth
showing; neither is derivable from the other, and inventing one from the other is how `agy-1`
ended up in the mock.

Two consequences that are not cosmetic:

- **Effort is part of the id, not a separate axis.** `agy` has eleven Gemini entries because each
  reasoning effort is its own model. There is no effort control to design — picking
  "Gemini 3.1 Pro (High)" *is* picking the effort.
- **The transcript stores the whole model, not a reference to it.** When a provider drops a model
  from its list, nothing can resolve its label any more, and an old turn would render as a bare id
  or an empty space. The label is recorded at the moment the turn ran.

Also settled here: **a provider is the CLI we drive, never the vendor of the model.** `agy` serves
Gemini, Claude *and* GPT models, so `Provider.routes` marks the distinction rather than letting the
provider's colour imply an owner.

## D-016 — `claude` is scoped with `--strict-mcp-config --setting-sources project`, and the daemon scrubs its environment

**2026-09-10.** Rejected: `--bare`, which this file and the blueprint both recorded as mandatory
until it was actually run.

`--bare` sounds like the scoping flag and is not one. Its own `--help`: *"Anthropic auth is strictly
`ANTHROPIC_API_KEY` or `apiKeyHelper` via `--settings` (OAuth and keychain are never read)."* The
owner signs in with a subscription, so `claude --bare -p` returns `Not logged in` in under a second.
Shipping that would have made claude unusable — a worse outcome than the leak it was meant to close.

The owner caught this by asking the obvious question: **Multica drives his Claude Code on this same
machine, so it must be possible.** It uses `--strict-mcp-config` (`server/pkg/agent/claude.go`).
Verified here 2026-09-10 with every `CLAUDE_*` and `ANTHROPIC_*` variable scrubbed:

| | Result |
|---|---|
| `"apiKeySource"` | `"none"` — and the turn ran, so OAuth was used |
| `"mcp_servers"` | `[]` — zero connected, against `clockify`/`square`/`shadcn` unscoped |
| `skills` | 72 by default, **17** with `--setting-sources project` |

**The environment must be scrubbed too, and this is the part that would have been missed.** Any
process spawned from inside a Claude Code session inherits `CLAUDECODE=1`,
`CLAUDE_CODE_MESSAGING_SOCKET` and friends, and a nested `claude -p` then hangs indefinitely — two
runs killed at 120s, with stdin both attached and closed. Scrubbed, the same command returns in
seconds. The daemon is exactly the kind of process someone will start from a terminal inside an
agent session, so this is not hypothetical.

**Where we differ from Multica deliberately:** it passes `--strict-mcp-config` only when an
agent-level MCP config exists, letting a task inherit the user's servers otherwise, and it denies
individual skills through a task-local `--settings` file. That suits a system where an agent is
configured per task. Ours is one chat window over whatever is installed, with no per-conversation
tool configuration to express intent — so scoping is unconditional, and inheriting the owner's
personal MCP servers is never the default.

## D-017 — The Protobuf decision (D-003) is deferred, not reversed

**2026-09-10.** D-003 chose Protobuf for the wire protocol and rejected hand-maintained JSON types.
Reading [Multica](../Reference/multica-main) properly undercut its main support: same architecture,
same old-daemon problem, and **zero `.proto` files in the repo** — Go structs with JSON tags, and
TypeScript hand-written to match, with no generation step anywhere.

Part of D-003's reasoning was already covered by D-002: server and daemon are one Go module, so
Go↔Go drift is a compile error with or without Protobuf. What Protobuf would actually buy is
Go↔TypeScript safety, and there are lighter ways to get that than adopting a schema language and a
codegen step before the first message exists.

So `buf` is **not installed** and the protocol starts as Go structs with JSON tags over WebSocket.
Revisit once real message shapes exist and the cost of hand-maintaining them is observable rather
than predicted. `sqlc` and `goose` were installed — Multica confirms sqlc; on goose we deliberately
diverge, since writing our own runner is ~900 lines of infrastructure before anything works.

## D-018 — The usage strip shows when a window resets, never how much is left

**2026-09-10.** Rejected: the percentage-and-capacity-bar readout the design shot showed and the
mock implemented (`claude 62% ▓▓▓░░ resets 4h 12m`).

That percentage does not exist. `claude`'s `rate_limit_event` — the only usage signal any of the
three providers emits — carries exactly this:

```json
{"status":"allowed","resetsAt":1789083000,"rateLimitType":"five_hour",
 "overageStatus":"rejected","isUsingOverage":false}
```

A status, a reset timestamp, a window name. Nothing about consumption. The 62% was invented for the
mock, survived the image round because a generated picture will draw whatever it is told, and was
still there when the surface was wired.

**Time remaining is not capacity remaining**, and a bar is read as the second. The elapsed fraction
of a five-hour window is computable, but showing it as a filled bar would state something we cannot
measure — the owner could be at 5% of his quota or 95% and it would look identical.

So the strip reads `claude · 5-hour resets 2h 25m`, counted down on the client so it stays true
without the server pushing. `codex` and `agy` keep "no limit data" against a dashed rule, which
remains a distinct state from "plenty left". The `--capacity-ok` / `--capacity-low` tokens and
`capacityClass()` are deleted: with no proportion, there is nothing to shade.

**This is the feasibility gate failing and then working.** The rule is "never design what the
backend can't deliver", and a design got approved with a number no provider emits. What caught it
was building the adapter — which is late, but not too late, and is why the backend step is allowed
to send a design back.

## D-019 — lucide-react is the icon family, and the only one

**Chosen 2026-09-10**, when the owner asked what family was in use and told us to lock on it.

Every icon in the app comes from `lucide-react`. There is no second icon package, and adding one
is the decision this entry exists to prevent: two families in one interface is visible at a glance,
because stroke weight, corner radius and optical size never match between them.

**The single exception is `components/chat/provider-icon.tsx`** — the claude, codex and agy marks.
Those are the vendors' own logos, which no icon family can supply. They are held to lucide's
conventions anyway: a 24×24 viewBox, `currentColor` rather than a brand hue, and sized by the same
`size-*` classes as every other icon, so they sit correctly beside real lucide glyphs.

**Rejected:** importing `@lobehub/icons` for the three brand marks. It is where the path data came
from (MIT), but the package pulls in 400+ transitive dependencies for three glyphs, so the paths are
inlined instead.

## D-020 — Only the daemon touches the filesystem

**Chosen 2026-09-10**, building the working-directory picker (B-3).

Directory browsing is a request the server forwards to the daemon and waits for, rather than an
`os.ReadDir` in the API handler.

The server is meant to run somewhere else — Coolify is the plan. Today it happens to share a machine
with the daemon, so reading the disk directly would appear to work perfectly, and would silently
start listing a *container's* directories the day it is deployed, presenting them as the owner's.
The bug would surface as a folder picker showing `/app` and `/usr`, long after anyone remembered
why.

This is the first exchange in the protocol that expects an answer; everything else is one-way. It
is matched by request id rather than by order (`hub.Ask`), because several pickers can be open at
once and the daemon may answer them in any sequence.

**Rejected:** an Electron-style native folder dialog, which is what Multica uses
(`apps/desktop/src/main/local-directory.ts`). It is not available to a web app, and would tie the
picker to the desktop shell that does not exist yet (L-2). What was taken from Multica instead is
its typed failure reasons — `not_found`, `not_a_directory`, `not_readable` — so the surface renders
its own wording rather than parsing a sentence.

## D-021 — The daemon owns each agent's whole process tree, not just the CLI

**Decided:** 2026-09-11, building the stop button (L-8)

Stopping a turn kills a Windows Job Object (or a Unix process group) that the CLI and everything it
spawns belong to, rather than the CLI process alone.

**Rejected: `exec.CommandContext`'s built-in cancel**, which kills the direct child. That is not a
stop. An agent mid-task is usually running something — a build, an install, an MCP server — and
those are children of the CLI, not of us. Verified rather than assumed: with a leader-only kill, a
`ping` the leader had spawned outlived it and went on holding the inherited stdout pipe, which is
also what wedges the parser
(`server/internal/agent/tree_windows_test.go:TestKillingOnlyTheLeaderLeavesTheGrandchildRunning`).

**Adopted from [Multica](../Reference/multica-main)** (`server/pkg/agent/proc_windows.go`), which
drives these same CLIs on this same machine, and carries an ordering that is easy to get wrong and
expensive to discover: the child is created **suspended**, assigned to the job, and only then
resumed. Windows grants job membership only to processes created *after* the assignment and never
retroactively, so assigning after a plain `Start` leaves a window in which the agent has already
spawned subprocesses outside the job. That is worse than owning nothing — the job then reports an
empty tree while the escaped processes run on, and a stop gets reported as confirmed when it is not.

Trimmed from Multica's version: its console handling (`CREATE_NEW_CONSOLE`) solves a popup problem
we do not have, because our daemon runs in a terminal its children inherit.

**Cost:** `golang.org/x/sys` becomes a direct dependency. Accepted over hand-rolled `LazyDLL`
bindings for the job-object calls, which is precisely the kind of code that fails silently and in
the wrong direction. Weighed against D-013's preference for few dependencies: x/sys is effectively
extended-stdlib, and the alternative here is worse code, not less code.

**Failure is degraded, not fatal.** If ownership cannot be taken the child is resumed and runs
unowned, with a warning naming the consequence — a stop that kills only the CLI. A launch that
failed outright would take the whole provider down over something environmental.

## D-022 — A stopped turn is a column on the entry, not a fourth entry role

**Decided:** 2026-09-11, building the stop button (L-8)

`entries.stopped boolean` rather than a `stopped` value in `entries.role`.

**Rejected: a new role.** The partial text lives on the agent entry, and a separate marker row would
put the note somewhere other than the thing it is about. `role` has a CHECK constraint precisely to
keep the set of things that can appear in a transcript small and meaningful, and "this turn ended
early" is a property of a turn rather than a new kind of event. This is the same call
[`Later.md`](Later.md) L-11 reaches for recording a folder move — a column rather than a row.

**Rejected: reusing `failure`.** They are different events and they read differently: a failure is
something going wrong, a stop is someone deciding they had seen enough. Rendering the second as the
first puts a red alert box around a deliberate act. The surface leads with the stop when both are
set, because a CLI's complaint on its way out is a consequence of the stop, not a reason for it.

## D-023 — A turn is bounded by silence, not by duration

**Decided:** 2026-09-11, closing the second half of B-8

A turn is ended when it has produced **nothing** for `TURN_IDLE_TIMEOUT` (default 15 minutes), not
when it has run for some total length of time.

**Rejected: a wall-clock cap per turn.** It kills a session that is working perfectly well and
merely taking a while, which on a coding agent is most real tasks. [Multica](../Reference/multica-main)
has a ticket for exactly this (MUL-3064) and ended up with the same answer: liveness belongs to an
inactivity watchdog, and a run that keeps emitting events is never killed for running long.

**Rejected: no timeout at all**, which is what we had. A CLI that wedges produces nothing and never
exits, and the turn then never ends — the composer stays locked and the indicator ticks against
nothing, through a refresh. Not hypothetical: a nested `claude -p` hanging indefinitely is already
recorded in D-016, killed by hand at 120 seconds.

**Why the budget is generous.** The asymmetry favours patience. A false positive throws away real
work and the quota spent earning it; a false negative only means waiting longer for a safety net
that exists for when nobody is watching — and since the stop button shipped (L-8), somebody who *is*
watching ends a turn in one click. Fifteen minutes is also set by the worst case rather than the
typical one: `codex` emits nothing between its session id and its finished answer, so for codex this
is effectively a cap on the whole turn. That is the limit of what the CLI tells us, and the number
is chosen to sit well clear of it.

**Ordering, when several things are true at once.** Stopped beats went-quiet beats the process
error, because a killed CLI's complaint on its way out is a consequence of the kill rather than a
reason for it, and the transcript should carry the truest account of why a turn ended.


---

## D-024 — A conversation with no name has no name

**Decided:** 2026-09-11, building automatic naming (L-9)

`conversations.title` is nullable, and NULL means nobody has named it. The name comes from the first
thing said in the conversation, taken once, and the placeholder lives in the UI.

**Rejected: a default string in the column** — which is what we had. `'Untitled conversation'` sat in
the database as though it were a title, so every layer had to treat a description as a name: search
matched it, a rename compared against it, and nothing could tell a conversation nobody had named
from one named that on purpose. It is a display decision that had leaked into storage.

**Rejected: a model call per conversation**, which is the version that names them *well*. It costs a
request, a wait, and a failure mode, per conversation, forever. The first message of a coding
conversation is nearly always a statement of the task, so the first line of it is right often
enough — and when it is wrong the owner renames it, which he could always do. The worst case is the
"Untitled conversation" we have now, with extra steps.

**Named once, never re-derived.** A name says where a conversation started, not where it went. A
sidebar whose rows rewrite themselves as a conversation drifts is one you cannot learn, and the
search already covers "where did it end up" by looking inside message bodies.

**The guard is in the statement, not the caller.** `WHERE title IS NULL` on the naming UPDATE, so a
name the owner typed can never be overwritten by one derived from a message, whatever order the two
arrive in. The API's own check is a cheap pre-filter over the top of it, not the rule.

**A message with no words in it names nothing.** A row of dashes, a bare code fence: better unnamed
than named something worse than nothing — and because unnamed is a real state rather than a magic
string, the next message can still name it.

## D-025 — Authentication lives in the Go server, not in Next.js middleware

The obvious place for a login in a Next.js app is middleware, and it is the wrong one here.

Middleware protects **pages**. The thing that has to be protected is the Go API on its own port,
and a browser that never loads a page can call it directly — `curl` against `:8080` does not pass
through Next.js at all. Putting the gate in middleware would produce an app that looks locked and
is not, which is worse than one that looks open, because nobody goes looking.

So the server is the authority and the web app is only a surface: `app/page.tsx` asks whether it is
signed in and renders accordingly, and every answer it gets is one the server independently
enforces on every request.

**Rejected: a reverse proxy doing basic auth.** It would work, and it moves the security boundary
into a config file that lives somewhere else and is edited by hand. The rule "every route needs the
owner, except these three" is worth having as code with tests around it.

## D-026 — Sessions are rows in Postgres, not signed tokens

A JWT is the default answer and it cannot be revoked. That is the whole decision.

Behind this login is a daemon that runs coding agents on the owner's machine. The question that
matters is not *is this token well-formed* but *should this token still work right now* — a laptop
left somewhere, a session on a phone that was sold. A signed token can only be answered by waiting
for it to expire. A row can be deleted, and "sign out everywhere" is one `DELETE`.

The cost is a database round trip per request, which for a single-user app on a local network is
not a cost.

**What is stored is the token's SHA-256, never the token**, so a database dump or a leaked backup
is a list of hashes rather than a list of working logins. SHA-256 rather than argon2 for this one:
the input is already 256 bits of randomness, so there is no dictionary to slow an attacker to, and
unlike the password this runs on every single request.

Two expiries, because they answer different questions: `expires_at` is how old a session may get
(a week), `last_seen_at` is how long it may go unused (two days). A session used daily for a month
is not the same risk as one abandoned in a browser tab.

## D-027 — The server refuses to start without its secrets

No default password, no `AUTH_DISABLED` switch, no development shortcut. `DAEMON_TOKEN` and
`WEB_ORIGIN` are both required, and the process exits naming the ones it is missing.

Every convenience considered here was a way for an unauthenticated server to reach production
quietly. A default password ships as the real one. An "auth off for local dev" flag gets set in a
Coolify environment at 2am. A server that will not boot is a loud, immediate, local failure; a
server that boots without a password is a silent, remote one, and the blast radius is a shell on
the owner's laptop.

The cost is that development needs the variables too, which the `Makefile` and `scripts/dev.ps1`
supply. That is one line in a file rather than a branch in the program — and it means there is
exactly **one** code path through authentication, which is the part that actually matters, because
the second path is the one nobody tests.

There used to be an `OWNER_PASSWORD_HASH` here too. It is gone: the password now lives on an
account row in Postgres (D-029), so there is no hash in any configuration to require, to mangle in
an env file, or to redeploy in order to change.

## D-028 — The daemon gets its own credential, not the owner's password

The daemon presents `DAEMON_TOKEN` as a bearer header on its websocket handshake; the browser
presents a session cookie. Two credentials, because they prove two different things: the password
is a person proving who they are, the token is one machine proving it is the one that was
installed.

Sharing the password would put it in plain text in a service configuration on a laptop, and
rotating it would sign the owner out of every device at the same time as re-authorising the
machine. They should be able to change independently, because they will need to.

The daemon socket is checked **before** the websocket upgrade, so an unauthorised dialler gets a
plain 401 rather than a working socket that is then closed — and nothing is registered in the hub
on the strength of a connection that is about to be rejected.

## D-029 — Accounts in Postgres, and one setup code to create the first one

**Superseded 2026-09-12 by D-032.** The users table and sessions stand; the setup code is gone.

Authentication began as a password hash in an environment variable. It worked, and it was the
wrong thing: signing in meant reading a runbook, and changing the password meant editing a Coolify
variable and redeploying. The owner's verdict was that from his side it was the worst part of the
app, and he was right — it was a deployment chore wearing a login's clothes.

So there is a `users` table: email (`citext`, unique), an argon2id hash, and sessions that belong
to a user. Signing in is an email and a password. Changing the password is a menu item.

**The first account is the hard part, and it is not an ordinary sign-up.** What is behind this
login runs coding agents on the owner's machine, so a sign-up form left open to whoever finds the
URL is a form that hands a stranger a shell. Four options were considered:

| | Why not |
|---|---|
| An invite code in configuration | Back to editing a variable to deploy — the thing being removed |
| Email confirmation | No mail server, and none wanted for one account |
| First request wins | A race with the internet, decided by who loads the page first |
| **A setup code printed at startup** | **Chosen** |

The server generates a random code per process, prints it while the app is unclaimed, and accepts
it once. Reading it requires the deployment's logs, and having those is the same thing as owning
the deployment — which is exactly the fact that needs proving. The owner is reading the deploy log
anyway, so it costs him nothing. The code is held in memory only: a restart kills it and prints a
new one, so a code that leaked into an old log is already dead.

Taken from Multica (`Reference/multica-main`), whose `multica login` proves a machine by having an
already-signed-in human approve it in a browser rather than by pasting a shared secret. Their
browser-approval flow for per-machine daemon tokens is the better version of this and is parked as
L-16; the setup code is the part that works before there is anybody to approve anything.

Once an account exists the endpoint answers 409 to the correct code, so sign-up closes permanently
rather than merely being hidden by the interface.

## D-030 — A websocket is authenticated once, so the hub has to be able to hang up

A session cookie is checked on every HTTP request. A websocket is checked once,
at the handshake, and then the connection lives as long as the tab does. So
deleting session rows — which is what signing out, signing out everywhere, and
changing a password all do — did not disconnect anything. A browser that had
been revoked kept receiving every conversation event, indefinitely.

That is worst in exactly the case the feature exists for. "Sign out everywhere"
is pressed because a device may be in the wrong hands; it deleted the rows,
reported success, and left that device's socket streaming.

Three ways to fix it were available:

| | |
|---|---|
| Re-check the session on every frame | Browsers only listen, so there are no incoming frames to hang the check on |
| Poll: revalidate each socket every N seconds | Simple, but leaves a window whose size is N, chosen by nobody for any reason |
| **Register the socket against its session and close it on revoke** | **Chosen** — deterministic, and the close happens in the same request that revoked |

So the hub stores who each socket belongs to (user, and session hash) rather
than a bare set of connections, and exposes `DisconnectUser` and
`DisconnectSession`. The API calls them **after** the rows are gone, never
instead: the database still decides whether a token is valid, and this only
stops a connection authorised before that decision from outliving it.

The close happens outside the hub's lock, because `Close` can block and holding
the lock through it would stall every other browser's events.

Found by adversarial review rather than by use, along with three other defects
in the same change (`docs/Bugs.md` B-15). Each has a test that fails without its
fix.

## D-031 — An account owns its conversations and its machine; workspaces will group work inside it

**2026-09-12.** Rejected: Multica's model, where a workspace owns everything and people are members
of it; and adding a personal workspace now so every row could point at one.

Invitation-only registration means more than one person on a deployment, and until now nothing said
whose a conversation was — the hub even sent every event to every browser (KnownGaps G-27, now
closed). So `conversations.user_id` is required, every statement a request can reach names the
account as well as the row, and somebody else's conversation answers exactly like one that never
existed. The hub holds one machine per account and has no broadcast to everyone at all: the only way
to send an event is to name whose it is.

Multica scopes everything to a workspace because its workspaces are shared by teams — issues,
agents and runtimes belong to the group, not the person. The owner's workspaces are different in
kind: separate areas a person creates inside their own account, with sharing and roles explicitly
out of scope. Ownership therefore stays with the account, and a workspace, when it arrives, is a
grouping column on top of it rather than a change of owner. Adding a personal workspace today would
have been a table and a foreign key that no screen uses (AGENTS.md rule 4), and removing it later if
the model changed would be the migration this avoids.

**If workspaces ever become shared** — invitations into a workspace, other members' conversations —
ownership has to move from account to workspace, and that is a real migration. That feature needs
its own spec first, so the trade is made knowingly.

The machine half is deliberately minimal until pairing (US2). The shared `DAEMON_TOKEN` proves "the
owner's machine", so it is attributed to the `OWNER_EMAIL` account at connection time and refused
until that account exists. A second account has no machine and is told so; it can never reach the
owner's.

## D-032 — Invitations in configuration, links by email, SMTP on the product's own domain

**2026-09-12.** Supersedes D-029's setup code. Rejected: keeping the setup code for the first
account; an invitation table with no screen to manage it; a transactional email vendor; six-digit
codes.

**Who may register** is `OWNER_EMAIL` plus `ALLOWED_EMAILS`, read at startup. Anybody else who tries
becomes one row in `access_requests` and one email to the owner, however often they ask; approving
means adding the address and restarting. A table would be the right home the day there is a screen
to edit it (L-18), and a table without one would mean approving by editing production data — the
exact chore this release removes. The owner registers through the same page as everyone else, which
is what retires the setup code: the first account is no longer special.

**Proving an address** is a link, not a code (design DD-003), stored like a session token — only
its SHA-256. `email_links` keeps `used_at` and `superseded_at` separately from expiry because the
page a dead link opens names which happened. Spending a link is one `UPDATE … WHERE used_at IS NULL
… RETURNING`, so two tabs cannot both succeed, and registering or resetting commits the link, the
account change and the new session in one transaction. The password is chosen after the link
(DD-002), so no unconfirmed account ever exists.

**Revealing nothing** is enforced in the handlers and held by tests: registering an address that
already has an account answers exactly like a new one (the email says it exists), and a reset
request answers identically either way, with the reset email sent in the background so response
time is not the oracle the text refuses to be.

**Sending** is SMTP to the Hostinger mailbox on `sparstrow.com`, TLS only (implicit on 465, required
STARTTLS otherwise). Mail from the domain it claims to come from is what deliverability mostly
depends on, and at invitation-only volume a vendor adds an account, a DNS change and a bill for no
difference the owner would see. The server refuses to start without complete mail settings; a
`log` transport exists for development and is refused whenever session cookies are Secure, because
a deployed log is no place for working links.

## D-033 — The daemon is its own installer, published unsigned until signing exists

**2026-09-13.** Rejected: a zip with a PowerShell install script (three files to extract, and
PowerShell's execution policy in the way of the one person who cannot be asked to change it); an MSI
or Inno Setup installer (a second toolchain and a second artifact to sign, for nothing a per-user copy
lacks); serving the download from the web image (a 7 MB binary in every web deploy); waiting for a
code-signing certificate (the owner chose to ship now and sign later).

**One executable.** Opened with no arguments, a release build copies itself to
`%LOCALAPPDATA%\Programs\sparstrowgen`, registers `sparstrowgen://` and a start-at-sign-in entry
for the current Windows user only, and starts itself with `run`. Opened by the browser with a pairing
link, it claims the request and restarts the background copy. No administrator rights, and US3's
updater has one file to replace.

**One server per build.** The release step bakes the API and websocket addresses in with `-ldflags`,
and a release ignores `SERVER_WS`, `SERVER_API` and `DAEMON_TOKEN`. Those variables are how the
development route works, and a leftover one would otherwise point an installed copy at a laptop or
connect it through the shared token without pairing. Development keeps its own data directory for the
same reason.

**Local, single, and quiet.** The credential lives in `%LOCALAPPDATA%` rather than the roaming
profile, because it identifies one computer. A named mutex keeps one background copy per Windows
user, and a named event lets a new install or pairing ask the old copy to stop (G-33). Windowless,
with a hidden console handed to the agent CLIs, as Multica does.

**Unsigned, and saying so.** Releases are GitHub release assets; the install page links to
`releases/latest/download/sparstrowgen-setup.exe` and tells the person that Windows will warn. Signing
is G-32.

## D-034 — Updates are a signed manifest, swapped in by the old version, and put back if the new one cannot reconnect

**Its Trust paragraph is replaced 2026-09-14 by D-035: there is no signing key.** Waiting for work,
rollback and the server's role stand.

**2026-09-13, US3.** Rejected: Windows code signing as the trust root (there is no certificate yet,
G-32, and a signature on an executable says nothing about which version is newest); trusting HTTPS to
GitHub alone (anyone able to publish a release or redirect a download could run code on every
computer); the server announcing versions (a second thing to deploy per release, and a compromised
server could push code); an updater framework such as Squirrel or MSI (a second toolchain, and none
waits for an agent CLI to finish); a Windows service as supervisor (it would run without the person's
PATH and agent credentials, which the release workflow rules out).

**Trust.** Each release publishes `sparstrowgen-update.json` — version, installer URL, SHA-256 — with a
detached ed25519 signature. The public key is built into every daemon; the private key stays on the
release machine, outside the repository ([runbook](runbooks/daemon-release.md)). A daemon reads no
field until the signature verifies, and keeps a download only if its SHA-256 matches. One package,
`internal/release`, is used by both the tool that signs and the daemon that verifies.

**Never interrupting work.** A check downloads. Installing waits until the turn registry is empty,
then closes to new turns under the lock turns start under, so work that begins before activation keeps
the update waiting however long it runs. A turn arriving in the instant after is refused with "send it
again in a moment". Automatic updates check a minute after starting and then hourly; turning them off
ends an automatic wait; Update now installs whatever the setting, still waiting for work.

**Rollback.** The running copy copies itself to `updates\sparstrowgen-updater.exe` and starts it, so the
swap is done by the version known to work. The updater waits for the old copy to exit, renames it
aside, copies the new one in and starts it. The new copy records its version once the server accepts
it; if that has not happened within two minutes, or it stops by itself, the updater kills it, puts the
old executable back, starts it, and leaves `result.json` saying why. The restored copy reports that
failure and does not retry that version by itself.

**The server decides nothing.** It stores the preference (on by default), relays Check now and Update
now to the computer, and shows what the computer reports. Hello carries `version`, `protocol` and
`selfUpdates`; their absence means a daemon older than this. `MinDaemonProtocol`, 0 today, is what
makes a computer too old: Chat then says to update it in Settings → Updates and sends no work. Raise
it only after a release those computers could update to.

## D-035 — Updates trust our GitHub releases and a checksum, as Multica's do; there is no signing key

**2026-09-14, owner, after US3 shipped.** Replaces D-034's Trust paragraph. His objection: a key that
someone must back up is a burden nobody using the app should carry, and losing it stops every computer
updating until each is reinstalled by hand. Multica has no key (`server/internal/cli/update.go`,
`.github/workflows/release.yml`): GitHub Actions builds each release from a tag and publishes
`checksums.txt`, and its daemon installs the latest release only if the download matches it.

**Trust.** A `daemon-vX.Y.Z` tag on a commit already on main makes
`.github/workflows/daemon-release.yml` check that the version is higher than the latest, run the daemon
and update tests on Windows, build, and publish the release as latest. Its `sparstrowgen-update.json`
names the version, the installer URL and the installer's SHA-256. The daemon reads it over HTTPS from
GitHub, refuses it unless the installer is on that same site, and keeps a download only if its SHA-256
matches (`internal/release`). Fields it does not know are ignored, so an older daemon reads a newer
manifest.

Rejected: keeping ed25519 with the key in GitHub's secret store (once GitHub builds releases the key
sits beside the upload rights, so a takeover gets both, and a lost or leaked key still means
reinstalling every computer); keeping the key on the owner's PC (the burden this removes). Accepted:
whoever controls the GitHub repository can ship an update to every computer. They could already
replace the installer the install page links to, and whoever controls our server can already send
agent work to a connected computer, so the key protected less than it appeared to. What protects
updates now is two-factor sign-in on every account with write access to the repository.

**Moving off the key.** 0.2.0 and 0.2.1 accept only a signed manifest, so 0.2.2 is published with a
`.sig` made by the old key ([runbook](runbooks/daemon-release.md)); releases after it carry none, and the
key is deleted once no computer needs it. A computer still on 0.2.0 or 0.2.1 after that reports that
it could not check, and needs the installer opened once. It keeps its pairing.

## D-036 — Each worktree is assigned its own local stack, from a registry outside the checkout

**2026-09-17, agent, building the Phase 2 prerequisites.** WORKFLOW.md requires isolated local
environments before `develop` opens to parallel work, and said only that an allocator must exist.
This is it.

**What was actually shared.** One Compose file with a fixed `container_name` and host port 5433, one
`.env`-free Makefile naming 5433/8080/3000, and — the dangerous one — one daemon home: a development
build keeps its credential in `sparstrowgen-dev`, so two worktrees were **one computer** as far as
the server was concerned, and a refused pairing in either deleted the credential both used.
`testdb` dialled 5433 too, so one agent's `go test` ran against whichever checkout owned that port.

**Assigned, not derived.** `devstack` gives a worktree the lowest free slot and records it in
`stacks.json` in the user's data directory. A derived value — hash the path, take a port — needs no
registry but cannot notice that something else on the machine already holds the port, and 5433 was
itself chosen to dodge a local Postgres on 5432. So each candidate port is probed before it is
handed out, and what was handed out is written down.

**The registry is outside the checkout** because its whole job is to stop two checkouts colliding,
which a file inside one of them could not do. It is not shared between machines and is not a secret:
it holds paths and port numbers.

**Slot 0 is held for the main checkout**, detected by `.git` being a directory rather than the file
a linked worktree has. Any other rule lets an agent's worktree take 5433/8080/3000 first, after
which the owner's next `make db` starts an empty database on another port — which from his side is
indistinguishable from his local data being deleted.

Rejected: Compose profiles (they separate services, not host ports); one shared Postgres with a
database per worktree (the port is only one of the collisions, and the daemon credential is the one
that actually broke things); telling agents to set the variables by hand (WORKFLOW.md already says
manually reusing defaults is not isolation, and a rule nobody can forget beats one everybody must
remember).