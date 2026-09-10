# AGENTS.md — workflow and engineering standard

Mandatory workflow, safety rules, and engineering standards for every AI coding agent working on
**sparstrowgen**.

---

## 1. How work happens here: design-driven development

A previous attempt at this app was plan-driven — a spec, then a plan, then tasks. The planning took
the time, the coding and testing didn't, and it never shipped. The order is now inverted, and this
is the single most important rule in this file.

**Two documents survive, and each has exactly one author.** The owner writes the spec: what he
wants and why, in his terms. The design answers what it looks like, and he picks it from rendered
options. Nothing else gets written before code.

### The loop

```
 1  Spec          what the owner wants and why, in his words — he approves it
 2  Feasibility   what can the backend actually deliver here?
 3  Options       2–3 genuinely different directions, rendered
 4  Pick          owner chooses one, or a mix
 5  Wire it       into the real app, on placeholder data
 6  Confirm       owner uses it in the app, not a mockup
 7  Contract      what the backend must provide, derived from the locked design
 8  Backend       built to that contract, nothing speculative
 9  Swap          placeholder data out, real data in
10  Verify        frontend-verify, against the real thing
```

**Invoke the `design-driven-feature` skill** for any feature work — it carries the procedure. This
section is the rule; that skill is the how.

### The spec says what; the design says what it looks like

Keep these from overlapping, because the failure is quiet. **A spec must not describe an
interface.** "A sidebar showing recent conversations" is a design decision smuggled into prose,
and it silently pre-empts the options the owner is supposed to choose between. A spec says what
someone needs to *do* and what is *true afterwards*; the design answers everything visual.

A spec also carries no technology — no tables, endpoints, or frameworks. `writing-specs` has the
procedure, and the short version is: **draft it from what the owner already said and hand it back
for correction**, rather than interviewing him for it.

**Skip the spec** for bug fixes, backend-only work, and small specific changes. It earns its place
when the owner has something to explain, not as a formality.

### The rule that makes it work

**Never design something the backend cannot deliver.** A screen showing data no provider emits is
waste: it looks finished, it gets approved, and then it cannot be served.

[`docs/Capabilities.md`](docs/Capabilities.md) is the feasibility surface — **read it before
designing anything.** When a design needs something not listed there, choose explicitly: check it
and add a verified row, redesign around it, or cut it to `Deferred.md` with a trigger. Never
"probably fine".

### Backend work does not start early

Steps 1–6 are cheap. Step 8 is expensive, and it does not begin until the owner has confirmed the
design in the real app. Building backend for a design that then changes is the second-biggest
waste after designing the undeliverable.

Backend-only work with no surface — the daemon connection, a migration, a protocol change — has
nothing to design. Build it.

---

## 2. Stack and layout

**`.sparstrowgen/blueprint.yaml` is the single source of truth for the stack, commands, and CLI
roster — read it, don't restate its facts here or anywhere else.** [README.md](README.md) carries
the folder layout and the architecture diagram. [`docs/Decisions.md`](docs/Decisions.md) carries
why each choice beat its alternatives.

Wiring detail:

- **Server and daemon are one Go module.** Both binaries build from `server/`, sharing
  `internal/`, so wire-protocol drift is a compile error rather than a runtime bug on a laptop
  running an older daemon.
- **The wire protocol is defined once**, in `proto/`, and generated into Go and TypeScript. Never
  hand-write a message shape on one side and mirror it by hand on the other.
- **Conversations are provider-neutral.** Our database is the transcript of record; a provider's
  own session is only a per-provider cache, because no agent CLI can resume another's session.
- **The daemon dials out only.** Nothing inbound to the owner's machine, no ports exposed.

### Hard constraints

These are not style preferences. Breaking one causes a class of bug rather than an ugly diff.

**Server state and client state stay separate.**

- TanStack Query owns anything fetched from the server — conversations, messages, runs, providers.
- Zustand owns client/view state — filters, drafts, modals, layout. Shared stores live in
  `packages/core`, never in `packages/views` or an app directory.
- Only auth stores may call the API client directly. Everything else goes through queries and
  mutations.
- Realtime events invalidate or patch the Query cache. They must **never** mirror server payloads
  into Zustand — that is two sources of truth for the same fact.
- Optimistic updates only when all of: the outcome is locally predictable, the user stays on the
  same screen, failure is rare, and rollback is trivial. Anything that navigates or confirms
  awaits the server first.
- Streaming a message uses a visible pending state with retry, not silent optimism.
- Zustand selectors must return stable references — never a freshly allocated object or array
  without a shallow comparison.

**Package boundaries.**

- `packages/core` — no `react-dom`, no direct `localStorage`, no UI libraries.
- `packages/ui` — no imports from `packages/core`, no business logic.
- `packages/views` — no `next/*`, no router imports. Navigate through the adapter, so the same
  component works in the desktop shell later.
- `apps/web` — the only place Next.js platform APIs belong.
- Every workspace declares its own direct dependencies in its own `package.json`.

**Protocol compatibility.** An installed daemon will one day be older than the server it talks to.
Protobuf handles the wire, but not the semantics:

- A server-driven enum switch needs a `default` branch.
- Don't pin a critical affordance to a single boolean; combine signals where you can.
- Parse anything crossing the boundary defensively and default missing fields deliberately.

`Reference/` holds read-only checkouts kept for architectural comparison — currently
[Multica](Reference/multica-main), a live working example of this same shape. **Consult it for
patterns before inventing new ones. Never edit anything under `Reference/`.** What was adopted
from it, adapted, and deliberately rejected is recorded as D-009 in
[`docs/Decisions.md`](docs/Decisions.md).

---

## 3. Git workflow

```
feature branch ──PR (squash)──► main
```

1. **Never edit `main` directly.** Work on `feature/<slug>` or `fix/<slug>`.
2. **One working directory per agent.** Two agents must never share a checkout — that is not a
   merge conflict resolved later, it is two processes writing the same files at once.
3. **Verification before the PR.** Typecheck and tests pass locally. For anything a browser can
   exercise, green tests are not sufficient — see §4.8.
4. **Commit and push without asking.** Once a coherent unit of work is done, commit it on the
   current branch and push. This file is the standing authorization. A commit that never leaves
   the local checkout is as unrecoverable as one never made.
5. **This does not authorize pushing to `main`**, opening a PR, or merging one.
6. **If a push is rejected**, don't force over it. Fetch, reconcile, push the result.

---

## 4. Engineering rules

**Rule zero: process serves shipping.** If a document isn't going to change what gets built, don't
write it. If a step exists because coding agents used to need hand-holding, skip it. When a rule
below and shipping genuinely conflict, say so out loud rather than quietly following the rule.

1. **Never guess code logic or file paths.** Inspect the authoritative file before writing code.
2. **Read the full error before diagnosing.** Un-truncated stack traces, actual log output.
3. **No superficial symptom patches.** No dummy fallbacks, silently swallowed exceptions, or
   commented-out tests. Fix the root cause.
4. **Never declare success without running verification.** Execute the check and read its output.
   If you skipped one, say which and why.
5. **Human-in-the-loop gates.** Destructive operations — dropping tables, deleting protected
   files, anything touching production — require explicit confirmation in chat.
6. **One feature at a time, all the way through.** Design, frontend, backend, real data, verified —
   then the next. Avoid over-engineering: minimal effective implementations, no speculative
   abstractions, no field the design doesn't show.
7. **Open questions block one thing, not everything.** Park it in `docs/OpenQuestions.md` and carry
   on with the rest. "Done except OQ-n" is a real, reportable state.
8. **End-to-end verification loop.** At the end of any feature or bug fix a browser can exercise,
   drive the real UI, report console errors and usability problems, fix them, and re-verify until
   clean. The `frontend-verify` skill is the concrete form — invoke it rather than improvising.
9. **All four states, on every surface.** Populated, empty, loading, error. The empty state is the
   one that gets skipped, and it is the first thing the owner sees on a feature they haven't used.
   Present before the owner confirms a design, not after.
10. **Check for a settings surface, every feature.** Does this introduce behaviour a user might
    reasonably want to configure? If yes, build that entry in the same PR. If not, it stays a
    straight feature — a required *check*, not a mandate to invent settings.
11. **Log a bug or a caveat in the same turn it surfaces**, whether the owner reports it or you
    notice it while doing something else. Wrong behaviour goes to `docs/Bugs.md`; something
    fragile, surprising, or half-finished that you deliberately left alone goes to
    `docs/KnownGaps.md` as a `caveat`. Going back to fix it is a separate decision — recording it
    is not optional. A problem mentioned only in chat does not exist to the next session.
12. **Shipping without proof is allowed. Shipping without saying so is not.** When a check can't be
    completed, name what you actually ran and open a `docs/KnownGaps.md` entry in the same change.

### Design rules

- **`DESIGN.md` is the doctrine.** Written with the owner by `design-brief`. Everything visual is
  accountable to it. Never restate its rules elsewhere — point at it; a duplicated doctrine keeps
  enforcing itself after the original changes.
- **`ai-design-slop` is loaded before writing UI**, so the tells never go in. It is a catalogue of
  what would be slop in *any* app; anything project-specific belongs to the doctrine.
- **No hardcoded colour, ever.** The doctrine is a theming contract, so a literal hue breaks every
  theme but the one you looked at.
- **`design-brief` and `design-system` run once**, when real UI work starts — not per feature.
  After that, building a screen means reading `DESIGN.md`, loading `ai-design-slop`, and writing
  the code.
- **Record why a design changed, not just what changed.** The reason usually generalises into a
  rule that stops the same debate recurring on every later page.
- **Mock data is named `*.mock.ts`.** A feature is not done while a shipped route still imports
  one — that makes leftovers greppable rather than something to remember.

---

## 5. Testing

No test infrastructure exists yet. These are the rules for when it does — they come from
[Multica](Reference/multica-main), which hit each of these problems at scale first.

### Never let a default test run a real agent CLI

**This is the one that matters most here.** We drive `claude`, `codex`, `agy` and `gemini`, and a
test that resolves one from `PATH` will spawn a real agent against the owner's authenticated
account and burn his quota — the exact limits this product exists to work around.

- Default tests pass a **test-created fake executable path**, or a path that deliberately does not
  exist. Never a real binary, never a bare command name.
- Real-agent smoke tests live behind a build tag and additionally check an environment variable
  before any executable lookup, so running the suite normally cannot reach them.
- Adding a new default agent command means adding it to the guard list, so ambient CLI execution
  fails the suite loudly rather than silently costing money.

### Where tests live

| What is tested | Where |
|---|---|
| Shared logic, stores, queries, hooks | `packages/core/*.test.ts` |
| Shared UI components, pages, forms | `packages/views/*.test.tsx` |
| Platform wiring — cookies, redirects, params | `apps/web/*.test.tsx` |
| Server, daemon, CLI adapters, protocol | Go tests beside the code |
| End-to-end flows | `e2e/*.spec.ts` |

Never test shared component behaviour in an app test file.

### One canonical layer per behaviour

Pure parsing, state transitions, and boundary matrices belong in a `.test.ts` beside the helper.
The component suite keeps the happy path, the wiring, accessibility, and named regressions — and
points at the canonical file in a comment. **Do not re-run a helper's matrix through a DOM mount.**

A `.test.ts` that needs no DOM starts with `// @vitest-environment node`. jsdom costs roughly
0.8s of setup per file and buys such a suite nothing. Do not add it to a test whose code branches
on `typeof window` — under node it would silently take the other path and still pass.

### Go tests build their rows through shared fixtures

Once there are DB-backed Go tests, they go through helpers in `server/internal/testutil`, not
open-coded `INSERT ... RETURNING id` with a matching cleanup, and not a
recorder/status-check/decode quartet per handler. Multica's `internal/handler` accumulated roughly
a thousand of the first and twelve hundred of the second before shared fixtures landed, and every
change to a shared contract then had to be made once per copy.

**A helper must never assert a product rule on a test's behalf.** A helper that knows what a
correct response looks like has taken the assertion away from the test making it. Keep an
assertion where the test wrote it when its message says something the shared one cannot.

### Writing them

- For a behavioural change, prefer writing the failing test in the correct package **before** the
  implementation.
- When adding an endpoint or a protocol message, add a malformed-input test alongside it.
- Anything a browser can exercise is verified in a browser as well (§4.8). A green suite is not
  evidence that a feature works.

### The verification ladder

Run the narrowest useful check while iterating; widen when the risk justifies it or when asked.
Commands are in `.sparstrowgen/blueprint.yaml`. The intended top rung is a single `make check`
running typecheck → unit tests → Go tests → end-to-end, so there is one command that means "all
of it" rather than four a person has to remember.

**Never claim a check passed unless you ran it**, and if you skipped one, say which and why.

## 6. Presenting options to the owner

The owner supplies scenarios and judgment; agents decide implementation. Don't hand library-level
choices back as open questions — recommend, explain why, let them veto. See [CLAUDE.md](CLAUDE.md)
for how they work and what depth to pitch at.

**If you can render it, render it.** For anything visual, seeing the options *is* the comparison —
say in one line what each direction optimises for, then let them look. Do not run the framework
below on a design.

For decisions with no picture — a protocol choice, two libraries, a tradeoff — each option carries:

- **Its own context** — what this option concretely *is*: what gets built, what they have to do
- **Its own scenario** — the question's scenario replayed under this option, so outcomes compare
  side by side. Same person, same moment, different result. This is the field that makes options
  answerable
- Pros and cons
- Score out of 10
- Blast radius if chosen wrong
- Caveats
- The agent's recommendation

A question with no options is not ready to be asked. Options describing *different* situations
from one another cannot be compared, and aren't ready either.

---

## 7. Database

- **PostgreSQL, self-hosted in Coolify.** sqlc + pgx for access, goose for migrations.
- **The schema is multi-provider from the first migration.** Provider switching is the product's
  reason to exist, and retrofitting it is the kind of rewrite this project avoids.
- **Vector search is pgvector, in the same Postgres.** Embeddings come from a separate HTTP
  service, never in-process.
- Single-user for now, so there is no tenancy boundary yet. Multi-user needs a deliberate design
  pass, not a column bolted onto existing tables.

---

## 8. Project memory

`docs/` holds everything that isn't code but must survive a session. **Read
[`docs/README.md`](docs/README.md) first.**

The load-bearing two:

- [`docs/Capabilities.md`](docs/Capabilities.md) — what the backend can deliver. Read before
  designing.
- [`docs/Decisions.md`](docs/Decisions.md) — load-bearing choices and their rejected alternatives.
  A few lines each, appended when a choice would be expensive to reverse.

Then the registers — `Bugs`, `KnownGaps`, `Deferred`, `OpenQuestions`, `Ideas` — each stating its
own format at the top. Only runbooks have a template, because only they are long enough to need
one.

When the owner says "park it", "later", or "just an idea", write it to the right file in the same
turn rather than relying on the conversation being re-read.
