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
