# AGENTS.md — workflow and engineering standard

Mandatory workflow, safety rules, and engineering standards for every AI coding agent working on
**sparstrowgen**.

---

## 1. Stack and layout

**`.sparstrowgen/blueprint.yaml` is the single source of truth for the stack, commands, and CLI
roster — read it, don't restate its facts here or anywhere else.** [README.md](README.md) carries
the folder layout and the architecture diagram. When the stack changes, change the blueprint; this
file only carries wiring detail the blueprint deliberately doesn't ("what tech are we on" lives
there, "how the pieces are wired" lives here).

Wiring detail:

- **Server and daemon are one Go module.** Both binaries build from `server/`, sharing
  `internal/`. This is deliberate: the wire protocol structs are used by both, so drift becomes a
  compile error rather than a runtime bug on a laptop running an older daemon.
- **The wire protocol is defined once**, in `proto/`, and generated into Go and TypeScript. Never
  hand-write a message shape on one side and mirror it by hand on the other.
- **Conversations are provider-neutral.** Our database is the transcript of record; a provider's
  own session is only a per-provider cache, because no agent CLI can resume another's session.
  Switching provider mid-conversation replays the history the target hasn't seen yet.
- **The daemon dials out only.** Nothing inbound to the user's machine, no ports exposed.

`Reference/` holds read-only checkouts kept for architectural comparison — currently
[Multica](Reference/multica-main), a live, working example of this same shape (Go server, local
daemon, Next.js front end, agents as first-class actors). **Consult it for patterns before
inventing new ones. Never edit anything under `Reference/`.**

---

## 2. Git workflow

```
feature branch ──PR (squash)──► main
     │
     └── typecheck + tests pass locally, before the PR
```

1. **Never edit `main` directly.** Work happens on `feature/<slug>`, `fix/<slug>`, or
   `task/<task-id>`.
2. **One working directory per agent, always.** Two agents — subagents, forked sessions, or
   separate windows — must never share a checkout. That is not a merge conflict resolved later; it
   is two processes writing the same files at once.
3. **Verification before the PR.** Typecheck and unit tests must pass locally. For anything a
   browser can exercise, green tests are not sufficient on their own — see §3.8.
4. **Commit and push without asking.** Once a coherent unit of work is complete, commit it on the
   current branch and push it. This file is the standing authorization. Commit at the end of a
   logical change, not per file touched. A commit that never leaves the local checkout is exactly
   as unrecoverable as one that was never made.
5. **This does not authorize pushing to `main`**, opening a PR, or merging one. Those stay
   explicit.
6. **If a push is rejected**, do not force-push over it. Fetch, reconcile, push the reconciled
   result.

When parallel agents start running on one feature at a time, this grows an integration-branch tier.
Not yet — a single-task branch targeting `main` is the right size for now.

---

## 3. Engineering rules

1. **Never guess code logic or file paths.** Inspect the authoritative file before writing code.
2. **Read the full error before diagnosing.** Un-truncated stack traces, actual log output. Base
   diagnoses on evidence, not on what the symptom resembles.
3. **No superficial symptom patches.** Do not mask errors with dummy fallbacks, silently swallowed
   exceptions, or commented-out tests. Fix the root cause.
4. **Never declare success without running verification.** Execute the check and read its output
   before claiming a task is complete. If you skipped a check, say which and why.
5. **Human-in-the-loop gates.** Destructive operations — dropping tables, deleting protected files,
   anything touching production — require explicit confirmation in chat.
6. **Micro-level, complete feature delivery.** Build one feature fully — data layer, server,
   protocol, UI, UX — before starting the next. If B depends on A, finish A completely, exposing
   the minimal clean interface B needs. Avoid over-engineering: minimal effective implementations,
   no speculative abstractions.
7. **Open question protocol.** An unanswered question blocks only the checklist item that depends
   on it — never the whole task, never the whole plan. Park it in `docs/OpenQuestions.md`, mark
   that one item `[~] blocked → OQ-n`, and complete everything else. "Done except OQ-n" is a real,
   reportable state. See §4 for how to present the question.
8. **End-to-end verification loop.** At the end of any feature or bug fix that a browser can
   exercise, drive the real UI, report console errors and usability problems, fix them, and
   re-verify. Loop until clean. The `frontend-verify` skill is the concrete form of this rule —
   invoke it rather than improvising.
9. **All four states, on every surface.** Populated, empty, loading, and error. The empty state is
   the one that gets skipped, and it is the first thing the owner sees on a feature he has not used
   yet. A surface shipped with only its populated state is not done, and saying "the happy path
   works" does not change that.
10. **Check for a settings surface, every feature.** Before calling a feature complete, ask whether
    it introduces behaviour a user might reasonably want to configure or set a default for. If yes,
    build that entry in the same PR. If it is a straight capability with no meaningful
    configuration, it stays a straight feature — this is a required *check*, not a mandate to
    invent settings (see rule 6).
11. **Document a bug or security issue in the same turn it surfaces**, whether the owner reports it
    or you notice it while doing something unrelated. A problem mentioned only in chat does not
    exist to the next session.
12. **Shipping without proof is allowed. Shipping without saying so is not.** When a check can't be
    completed, name what you actually ran and open a `docs/KnownGaps.md` entry in the same change.
    Never tick a box on weaker evidence than it asked for and stay silent — a ticked box that
    quietly means "looked right to me" devalues every other ticked box in the repo.

### The design skill chain

`design-brief` → `design-system` → `interactive-prototype` → `ai-design-slop` → `frontend-verify`
→ `slop-audit`.

- `design-brief` writes the doctrine by interviewing the owner. Nothing downstream runs before it
  exists, and everything downstream is accountable to it.
- **Never restate the doctrine's rules inside another skill or checklist. Point at it.** A
  duplicated doctrine keeps enforcing itself after the original changes.
- `ai-design-slop` is a catalogue of tells that would be slop in *any* app — portable and
  deliberately free of this project's tokens. Anything project-specific belongs to the doctrine.
- `slop-audit` is **report-only**, and an author auditing their own surface is not a second
  opinion. Build with `ai-design-slop` loaded, then audit separately.
- **Record why a design changed, not just what changed.** The reason usually generalises into a
  rule that stops the same debate recurring on every later page.

---

## 4. Presenting decisions to the owner

The owner supplies user scenarios and the experience expected; agents decide the implementation.
Don't hand library-level choices back as open questions — recommend, explain why, let him veto.
See [CLAUDE.md](CLAUDE.md) for how he works and what depth to pitch at.

When something genuinely is his call, every option carries:

- **Its own context** — what this option concretely *is*: what gets built, what he has to do
- **Its own user scenario** — the question's scenario replayed under this option, so outcomes
  compare side by side. Same person, same moment, different result. This is the field that makes
  options answerable
- Pros and cons
- Score out of 10
- Blast radius if chosen wrong
- Caveats
- The agent's recommendation

A question with no options is not ready to be asked. Options that describe *different* situations
from each other are not comparable and are not ready either.

---

## 5. Database

- **PostgreSQL, self-hosted in Coolify.** Access through sqlc + pgx; migrations through goose. See
  the blueprint for versions.
- **The schema is multi-provider from the first migration.** Provider switching is the product's
  whole reason to exist, and retrofitting it into a single-provider schema is exactly the kind of
  rewrite this project is trying to avoid.
- **Vector search is pgvector, in the same Postgres.** Embeddings are computed by a separate HTTP
  service and never in-process — that keeps the backend language decision free of the ML
  ecosystem, and keeps retrieval out of a cold-start path.
- Single-user for now, so there is no row-level tenancy boundary yet. When multi-user arrives it
  needs a deliberate design pass, not a column bolted onto existing tables.

---

## 6. Project memory

All non-code project memory lives in `docs/`. **Read [`docs/README.md`](docs/README.md) first** —
it holds the lifecycle (idea → spec → owner review → plan → tasks → code), the register files, and
the table mapping "what situation am I in" to "which file does this go in".

Every file type has a skeleton in [`docs/templates/`](docs/templates/). Copy the matching one
rather than inventing a shape; they encode the sections that make "done" mean the same thing every
time it is written.

When the owner says "park it", "later", or "just an idea", write it to the right file in the same
turn rather than relying on the conversation being re-read.
