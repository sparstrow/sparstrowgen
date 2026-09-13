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
(§5, and the deploy runbook's daemon-token step). To sign in, use **Forgot your password?** and read
the reset link straight from the mailbox; there is no need to remember or store a password at all.

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
7. **Log a bug or a caveat in the turn it surfaces.** Wrong behaviour → [`docs/Bugs.md`](docs/Bugs.md).
   Something fragile or surprising you deliberately left alone → [`docs/KnownGaps.md`](docs/KnownGaps.md).
   Fixing it is a separate decision; recording it isn't optional.
8. **Shipping without proof is allowed. Shipping without saying so is not.** Name what you ran, and
   open a KnownGaps entry in the same change.
9. **An open question blocks one thing, not everything.** Park it in `docs/Later.md`, build the
   rest, report "done except L-n".
10. **Destructive operations need confirmation in chat** — dropping tables, deleting protected
    files, anything touching production.

### Design

- **`DESIGN.md` is the doctrine** (written with him by `design-brief`). Point at it; never restate
  its rules elsewhere, or the copy keeps enforcing itself after the original changes.
- **Load `ai-design-slop` before writing UI**, so the tells never go in.
- **No hardcoded colour, ever.** The doctrine is a theming contract; a literal hue breaks every
  theme but the one you looked at.
- **Images pick a direction; the prototype makes the decision.** Say so when showing
  shots — an image has no interaction, no real data, and no states, and if he thinks
  approving one approved the design, the prototype step feels like re-opening it.
- **`design-brief` and `design-system` run once**, when real UI work starts — not per feature.
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
commit that never leaves the checkout is as unrecoverable as one never made. That does *not* extend
to pushing `main`, opening a PR, or merging. If a push is rejected, fetch and reconcile rather than
forcing.

Conventional prefixes: `feat(scope)`, `fix(scope)`, `refactor(scope)`, `docs`, `test`, `chore`.

---

## 6. Where things go

[`docs/README.md`](docs/README.md) has the full map. The short version:

| | |
|---|---|
| Can the backend do this? | [`Capabilities.md`](docs/Capabilities.md) — if it's unanswered, answer it there |
| Why we chose X over Y | [`Decisions.md`](docs/Decisions.md) |
| Built but unproved, or a caveat noticed in passing | [`KnownGaps.md`](docs/KnownGaps.md) |
| Behaving wrong | [`Bugs.md`](docs/Bugs.md) |
| Question, parked, or just an idea | [`Later.md`](docs/Later.md) |
| What he wants and why | [`docs/specs/`](docs/specs/) |
| Several changes at once, after he looks at something built | [`docs/feedback/`](docs/feedback/) |
| Only a human can do it — dashboard, DNS, secrets | [`docs/runbooks/`](docs/runbooks/) |
| Which branch/environment/release path is active | [`WORKFLOW.md`](WORKFLOW.md) |

Skills carry procedure so this file doesn't: `design-driven-feature`, `writing-specs`,
`design-shots`, `interactive-prototype`, `frontend-verify`, `ai-design-slop`, `testing`,
`design-brief`, `design-system`, `feedback-round`.

When he says "park it", "later", or "just an idea", write it down in the same turn. Chat is not read
by the next session.

**When he starts numbering feedback, invoke `feedback-round` and stop building.** Capture every item
verbatim as it arrives, triage the lot once he says he's finished, then work. Fixing item 1 while
item 4 is still coming wastes both — his list goes unfinished, and item 4 often changes what item 1
meant.
