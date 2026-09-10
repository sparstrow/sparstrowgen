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

This surfaced immediately: `gemini` is installed on the owner's machine but he has no account
signed in. It is **blocked**, not "not built" — a real, permanent-until-fixed state a provider
picker UI must show with a reason, the way Multica's `RuntimeUnusableNotice` does, not silently
omit the way an unbuilt provider is omitted. See [`Capabilities.md`](Capabilities.md) and
[`Later.md`](Later.md) L-6.

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
