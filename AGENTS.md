# How we work on sparstrowgen

> **`CLAUDE.md` and `AGENTS.md` are the same file.** Different agents read different names, and
> both must be followed. Edit one, copy it to the other — never let them drift.

---

## 1. The owner

Business systems analyst — Dynamics NAV and EDI, and he spends his days advising departments on new
procedures and walking them through process changes.

| Speak freely | Explain plainly, with the tradeoff |
| --- | --- |
| APIs, client/server, how they communicate | Specific library choices |
| SQL, general data modelling | Embeddings and vector search |
| App structure, product and UX patterns | Infrastructure and database internals |

Strong product intuition from having used a lot of apps, even where implementation knowledge is
thinner. Never condescend, never assume.

**He supplies scenarios and judgment; you decide implementation.** Don't hand library-level choices
back as questions — recommend, explain why, let him veto.

**He works visually.** He calls it *vibe coding*: ask for something, see designs, pick one, watch it
get wired into the real app, then have the backend built underneath. Design decisions he makes
himself, from options. Backend decisions he wants decided well on his behalf.

**Long-term correct over fast** — he won't build something knowing it will be rewritten. That
applies to schema, protocol, and boundaries. It does *not* mean more documents: two attempts at
this app now, and the first died of planning. Spend care on engineering, not paperwork.

### The agent's testing account

`agent@sparstrow.com` is a real Hostinger mailbox the owner gave the agent full API access to, and
now also a real sparstrowgen account on production, invited on purpose so account, registration and
email flows can be tested end-to-end without asking him for anything (closed
[`docs/KnownGaps.md`](docs/KnownGaps.md) G-29). Use it freely for that.

Its password is never written to the repo, chat, or a commit — the same rule as any other secret
(§5, and the deploy runbook's daemon-token step). **The agent never types a password or creates an
account**, not even this one and not when asked. When a test needs it signed in, ask him to sign it in
once in the Browser pane; the session lasts a week. Verification and reset links are read straight
from the mailbox.

**Testing on production with it** (`app.sparstrow.com`), so a bug he reports gets reproduced there
before it is fixed, and the fix is seen there after it deploys:

- **Give it its own computer.** Build `cmd/daemon`, and run it with `SPARSTROWGEN_HOME` set to a
  scratch folder, `SERVER_WS=wss://api.sparstrow.com/daemon` and `SERVER_API=https://api.sparstrow.com`.
  Create the pairing from the signed-in pane (`POST /api/machines/pairings`), run
  `pair -request <request>`, approve it (`POST /api/machines/pairings/{id}/approve`), then start it.
  Without its own `SPARSTROWGEN_HOME` it reads his computer's credential, and a refusal deletes it.
- **Never use his computer for a test.** Do not stop, restart or replace his installed daemon without
  asking. A copy started from an agent's shell loses his Windows user environment (docs/Bugs.md B-28).
- **Never run `claude` itself from an agent shell.** Inside the Claude desktop app that shell carries
  the app's own sign-in variables, and one run rewrote his CLI's credentials (docs/Bugs.md B-53). Go
  through a test daemon, which strips them, or strip every `CLAUDE*` and `ANTHROPIC_*` variable first.
- **Keep it small.** Conversations go in a scratch folder, and every turn spends his real agent quota,
  so prompts stay short. To reproduce the old behaviour, build the daemon from `main` in a scratch copy.
- **Clean up.** Disconnect the test computer (`DELETE /api/machines/{id}`) and stop its process when
  done. Keep the account.

---

## 2. How features get built

```
Spec         what he wants and why, in his words. He approves it.
Feasibility  can the backend actually deliver it?
Shots        2–4 directions as generated IMAGES. He picks a direction. ~2 min each.
Design       the chosen direction, clickable. He picks the details.
Wire         into the real app on mock data. He confirms it there.
Backend      built to the locked design's contract. Mocks swapped out.
Verify       frontend-verify, against real data.
```

**Each step is cheaper than the one after it**, so the expensive step only runs on
something already wanted. Images cost two minutes, so four directions get looked
at; coding prototypes caps that at two in practice, and the second is usually a
variation on the first.

**Invoke `design-driven-feature`** for any feature — it carries the procedure.

**Never design something the backend can't deliver.** A screen showing data no provider emits is
waste: it looks finished, it gets approved, then it can't be served.
[`docs/Capabilities.md`](docs/Capabilities.md) says what's possible — read it before designing.
When a design needs something not listed, choose explicitly: verify it and add a row, redesign
around it, or park it in [`docs/Later.md`](docs/Later.md). Never "probably fine".

**The spec says what; the design says what it looks like.** A spec carries no technology and **no
interface description** — "a sidebar showing recent conversations" is a design decision smuggled
into prose, and it pre-empts the options he's meant to choose between. Say what someone needs to
*do* and what's *true afterwards*. Draft it from what he already said and hand it back for
correction, rather than interviewing him.

**Skip the spec** for bug fixes, backend-only work, and small specific changes. "Make that button
secondary" doesn't need three directions either — read the room.

**Backend work starts last.** Building for a design that then changes is the second-biggest waste
after designing the undeliverable. Backend-only work with no surface has nothing to design — just
build it.

---

## 3. Stack and layout

**[`.sparstrowgen/blueprint.yaml`](.sparstrowgen/blueprint.yaml) is the single source of truth for
the stack and commands** — read it, don't restate it. [README.md](README.md) has the folder layout;
[`docs/Decisions.md`](docs/Decisions.md) has why each choice beat its alternatives.

- **Server and daemon are one Go module.** Both build from `server/`, sharing `internal/`, so
  wire-protocol drift is a compile error rather than a runtime bug on an older daemon.
- **The wire protocol is defined once** in `proto/` and generated into Go and TypeScript. Never
  hand-mirror a message shape on the other side.
- **Conversations are provider-neutral.** Our database is the transcript of record; a provider's
  session is only a cache, because no agent CLI can resume another's.
- **The daemon dials out only.** Nothing inbound to his machine.

### Hard constraints

Breaking one of these causes a class of bug, not an ugly diff.

**Server state ≠ client state.** TanStack Query owns anything from the server. Zustand owns view
state — filters, drafts, modals — with shared stores in `packages/core`. Realtime events invalidate
or patch the Query cache and must **never** mirror payloads into Zustand. Optimistic updates only
when the outcome is predictable, the user stays put, failure is rare, and rollback is trivial;
anything that navigates awaits the server. Selectors return stable references.

**Package boundaries.** `packages/core`: no `react-dom`, no direct `localStorage`, no UI libraries.
`packages/ui`: no `core` imports, no business logic. `packages/views`: no `next/*`, no router
imports — navigate through the adapter so the desktop shell stays possible. `apps/web`: the only
home for Next.js platform APIs.

**An installed daemon will one day be older than the server.** Enum switches need a `default`
branch, don't pin an affordance to one boolean, and default missing fields deliberately.

`Reference/` holds read-only checkouts for comparison — currently
[Multica](Reference/multica-main), a live system of this same shape. **Consult it before inventing
patterns. Never edit anything under it.** What was adopted and overridden is D-009 in Decisions.

---

## 4. Rules

**Rule zero: process serves shipping.** If a document won't change what gets built, don't write it.
When a rule below and shipping genuinely conflict, say so out loud rather than quietly following
the rule.

1. **Never guess code or paths** — open the file. Read the full untruncated error before diagnosing.
2. **No symptom patches.** No dummy fallbacks, swallowed exceptions, or commented-out tests.
3. **Never claim a check you didn't run.** If you skipped one, say which and why.
4. **One feature at a time, all the way through.** No speculative field, endpoint, or abstraction —
   nothing the design doesn't show.
5. **All four states on every surface** — populated, empty, loading, error. Present *before* he
   confirms a design; the empty state is the one he'll see first on a feature he hasn't used.
6. **Anything a browser can exercise gets verified in a browser** (`frontend-verify`). A green
   typecheck is not evidence a feature works.
7. **A bug you find gets fixed in the turn it surfaces, and logged as fixed.** Wrong behaviour →
   [`docs/Bugs.md`](docs/Bugs.md): what was wrong, what fixed it, and a one-line release note in plain
   words, so the change log and release announcements are written from that file. **It stays open only
   when fixing it needs the owner** — his decision, his access, his computer. Then log it, put the
   question in [`docs/Later.md`](docs/Later.md) with your recommendation, and carry on with other
   work; he answers there later. Something fragile or surprising you deliberately left alone →
   [`docs/KnownGaps.md`](docs/KnownGaps.md). Recording is never optional (owner, 2026-09-14).
8. **Shipping without proof is allowed. Shipping without saying so is not.** Name what you ran, and
   open a KnownGaps entry in the same change. Every check you could not run goes into
   [`docs/Unverified.md`](docs/Unverified.md) as a step someone can take, with who can run it.
   **Verify when time permits:** before finishing a turn, run any open check the agent can run
   itself, and hand the owner the ones only he can run while he is testing anyway. Mark each
   verified with the date and the proof, or failed with a bug — never on weaker evidence than it
   asks for.
9. **An open question blocks one thing, not everything.** Park it in `docs/Later.md`, build the
   rest, report "done except L-n".
10. **Destructive operations need confirmation in chat** — dropping tables, deleting protected
    files, anything touching production.

### Design

The design system lives in Claude, not in this repo:
[sparstrowgen Design System](https://claude.ai/artifact/Lyougp9xy958FaBcXWQFyS) — private, so open it
with the Artifact tool. Read its `project/README.md` first, then the README of any component before
using it; `project/tokens.json` has every token and its usage note. Agents that cannot open it
(Codex, agy) ask him to paste the part they need. Point at it; do not copy its rules into skills,
comments or docs, or the copy keeps enforcing itself after the original changes.

- **Build from what exists.** Use a primitive from the system before writing one, and check the
  shadcn registry before hand-building a missing one. A new persistent surface or navigation
  destination is his call, shown to him first.
- **No hardcoded colour, ever.** Colour resolves through a token, so every appearance works, not just
  the one you looked at.
- **Keep the system true to the code.** The system is built from `apps/web` (`globals.css`,
  `themes.css`, `components/ui`, `components/chat`). When a token or component changes there,
  update the artifact in the same turn, or record the drift in `docs/Unverified.md`. Where they
  disagree, the code is what shipped.
- **Load `ai-design-slop` before writing UI**, so the tells never go in.
- **Images pick a direction; the prototype makes the decision.** Say so when showing shots — an image
  has no interaction, no real data, and no states, and if he thinks approving one approved the
  design, the prototype step feels like re-opening it. Shots and prototypes live in `docs/design/`.
- **Record his design reactions** in `docs/Decisions.md` the turn he gives them, with the reason —
  the reason is what carries over to the next screen.
- **Mock data is `*.mock.ts`.** A feature isn't done while a shipped route imports one.

### Presenting a choice

**If you can render it, render it** — say in a line what each direction optimises for, then let him
look. For choices with no picture: what each option *is*, what happens under it, the tradeoff, and
your recommendation. Four things, not a form.

---

## 5. Git

Read [`WORKFLOW.md`](WORKFLOW.md) before choosing a target branch or changing release automation.
Its **Active phase** is authoritative. While it says Phase 1, the path remains feature branch → PR
(squash) → `main`. Phase 2's `develop` → `staging` → `main` promotion protocol does not begin until
the owner completes its activation gate. Never edit a protected branch directly; never share a
checkout between two agents.

**Commit and push your branch without asking** — this file is the standing authorization, and a
commit that never leaves the checkout is as unrecoverable as one never made. **Opening a PR and
squash-merging it into `main` is authorized too** (owner, 2026-09-14): once its checks are run and
recorded in the PR, merge without asking. Pushing `main` directly is still never done. If a push is
rejected, fetch and reconcile rather than forcing.

Conventional prefixes: `feat(scope)`, `fix(scope)`, `refactor(scope)`, `docs`, `test`, `chore`.

---

## 6. Where things go

[`docs/README.md`](docs/README.md) has the full map. The short version:

| | |
|---|---|
| Can the backend do this? | [`Capabilities.md`](docs/Capabilities.md) — if it's unanswered, answer it there |
| Why we chose X over Y | [`Decisions.md`](docs/Decisions.md) |
| Built but unproved, or a caveat noticed in passing | [`KnownGaps.md`](docs/KnownGaps.md) |
| A check nobody has run yet — verify when time permits | [`Unverified.md`](docs/Unverified.md) |
| Behaving wrong | [`Bugs.md`](docs/Bugs.md) |
| Question, parked, or just an idea | [`Later.md`](docs/Later.md) |
| What he wants and why | [`docs/specs/`](docs/specs/) |
| Several changes at once, after he looks at something built | [`docs/feedback/`](docs/feedback/) |
| Image directions and clickable prototypes, before they are built | [`docs/design/`](docs/design/) |
| Only a human can do it — dashboard, DNS, secrets | [`docs/runbooks/`](docs/runbooks/) |
| Which branch/environment/release path is active | [`WORKFLOW.md`](WORKFLOW.md) |

Skills carry procedure so this file doesn't: `design-driven-feature`, `writing-specs`,
`design-shots`, `interactive-prototype`, `frontend-verify`, `ai-design-slop`, `testing`,
`feedback-round`.

When he says "park it", "later", or "just an idea", write it down in the same turn. Chat is not read
by the next session.

**When he starts numbering feedback, invoke `feedback-round` and stop building.** Capture every item
verbatim as it arrives, triage the lot once he says he's finished, then work. Fixing item 1 while
item 4 is still coming wastes both — his list goes unfinished, and item 4 often changes what item 1
meant.
