# docs/

Working memory for the project. Everything that isn't code but needs to survive
a session lives here.

```
docs/
├── templates/                skeletons for every file type below
│   └── README.md             ← start here when creating any new document
├── specs/                    what the owner wants, in the owner's terms —
│   └── README.md             user stories, written BEFORE any plan
├── plans/                    approved plans — the technical "how"
├── tasks/                    executable specs — the "how"
│   ├── MasterTaskQueue.md    global run order + concurrency tags (active bands only)
│   ├── CompletedMasterQueue.md  fully-done bands, archived out to keep the above short
│   └── <phase>/              phase spec + individual tasks
├── runbooks/                 manual steps only a human can do (external
│   ├── README.md             ← start here: the owner's action-item checklist
│   └── <topic>.md            dashboards, OAuth apps, anything an agent
│                             shouldn't act on your behalf for). Not a
│                             lifecycle stage — these don't graduate into
│                             code, they just sit here as reference.
├── research/                  findings from outside this repo — prior art,
│                             comparable systems, library behaviour. Reference,
│                             not a lifecycle stage: nothing graduates out of
│                             here, but entries feed specs and ideas. Say how
│                             strong the evidence is — read source vs. summary.
├── feedback/                  raw owner/user reaction, not yet triaged
│   └── README.md             ← format, workflow, index
├── bug/                       owner-reported or agent-found wrong behavior
│   └── README.md             ← format, workflow, index
├── security/                  vulnerabilities, trust-boundary violations
│   └── README.md             ← stricter format, index
├── OpenQuestions.md          decisions waiting on the owner
├── Deferred.md               agreed to build, explicitly parked
├── KnownGaps.md              built, but not verified — or verified to be limited
└── Ideas.md                  unscoped — might never be built
```

## Lifecycle

```
idea ──────────────► Ideas.md
  │                  NOT a one-line stub. What is true in the code today,
  │                  the reframe, an arguable shape, the decisions it needs
  │                  — answered nowhere. Skill: `elaborating-ideas`
  │ (owner picks it up)
  ▼
spec ──────────────► docs/specs/<date>-<slug>.md
  │                  user stories, acceptance scenarios, what the interface
  │                  should feel like. NO technology.
  │ (owner reviews — the cheapest point to catch a wrong direction)
  ▼
plan ──────────────► docs/plans/<date>-<slug>.md
  │                  the technical "how". Splits the spec into foundational
  │                  work and per-story work. Links the spec, never restates it.
  │                  open decisions go to OpenQuestions.md until answered
  │ (owner approves)
  ▼
task ──────────────► docs/tasks/<phase>/T-<id>-<slug>.md
  │                  MUST contain zero open questions
  │                  each carries a Serves row: a user story, or the story
  │                  phase it unblocks
  ▼
code ──────────────► anything parked mid-flight goes to Deferred.md
                     anything shipped-but-unproved goes to KnownGaps.md
```

**Internal work skips the spec.** Anything that only changes how the repo is
built, checked, documented, or governed goes straight to a plan whose **Spec**
row reads `n/a (internal)`. Anything the owner can see, use, or reach starts
with a spec. When it's genuinely unclear, ask.

## The rule that matters

**A task document must be executable without asking the owner anything.** Every
decision it needs is either already made in its plan, or made and recorded inside
the task itself.

This is the whole point of splitting plans from tasks: plans are where
uncertainty is allowed, tasks are where it isn't.

### One blocked piece doesn't stop the task

If converting a plan into tasks surfaces a genuine question, it goes to
`OpenQuestions.md` — and **only the checklist item that depends on it waits**:

```markdown
- [x] Batch writes to Postgres
- [~] Auto-commit dirty tree before yielding   ← blocked → OQ-1
- [x] Replay buffer oldest-seq first
```

Everything else in that task still gets built and ticked off. The task is
reported as **done except OQ-1** — a real, closeable state, not a stalled one.
One missing piece must not stop the plate being served.

When the question is answered: unblock the item, finish it, and delete the entry
from `OpenQuestions.md`.

### Shipping without proof is allowed. Shipping without *saying so* is not

Verification sometimes cannot be completed — a platform won't deliver the signal,
the surface that exercises the code doesn't exist yet, the harness can't reach it.
That is a normal outcome, and it is not a reason to hold a change back.

It **is** a reason to write it down. Whenever a checklist item is ticked on
weaker evidence than it asked for:

1. Say so in the task's Result section, naming what was actually run.
2. Open an entry in [`KnownGaps.md`](KnownGaps.md) **in the same change**.
3. Name the phase or task that should close it, if one is obvious.

The rule this protects: *"done" must mean the same thing every time it is
written.* A ticked box that quietly means "looked right to me" devalues every
other ticked box in the repo, and the next agent has no way to tell which is
which. A caveat that lives only in a chat message does not exist — chat is not
read by the agent who picks this up in three weeks.

## Which file does this go in?

| Situation | File |
|---|---|
| "Let's do that later" | `Deferred.md` |
| "I'm not answering that right now" | `OpenQuestions.md` |
| "It's built, but I couldn't prove it works" | `KnownGaps.md` |
| "It works, but only within these limits" | `KnownGaps.md` |
| "Might be nice one day" | `Ideas.md` |
| "Here's how I want to use it, and what it should feel like" | `specs/` |
| "Here's how we'll build what the spec asks for" | `plans/` |
| "Here's exactly how, step by step" | `tasks/` |
| "Only a human can do this part (external dashboard, OAuth app, secrets)" | `runbooks/` |
| "This is behaving wrong" — owner-reported or agent-found | `bug/` |
| "This is a vulnerability / trust-boundary issue" — owner-reported or agent-found | `security/` |
| "Here's some raw feedback — not sure yet if it's a bug, an idea, or a real feature" | `feedback/` |

Once you know the destination, [`templates/`](templates/README.md) has the
skeleton for it — plans, phase specs, tasks, verification tasks, bugs,
security reports, runbooks, and entries for all four registers. Copy, fill in,
delete the guidance comments.

## Open questions must carry options

Per `AGENTS.md` §4, every entry in `OpenQuestions.md` needs full context, a
plain user-side scenario, and concrete options. Each option carries:

- Its own context — what this option *is*, concretely
- Its own user scenario — the question's scenario replayed under this option
- Pros and cons
- Score out of 10
- Blast radius if chosen wrong
- Caveats
- The agent's recommendation

A question with no options is not ready to be asked. Options that describe
*different* situations from each other are not ready either — replaying one
shared moment is what lets the owner compare them.

## Answered questions

When an open question gets resolved — including implicitly, because a later
decision settles it — the answer is recorded in the plan or task that consumed
it and **the entry is deleted from `OpenQuestions.md`**. That file only ever
holds what is still open, so its length is a real signal.
