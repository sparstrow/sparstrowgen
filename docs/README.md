# docs/

Working memory for the project. Everything that isn't code but needs to survive a session.

```
docs/
├── templates/          skeletons — spec, plan, runbook
├── specs/              what the owner wants, in the owner's terms. No technology.
├── plans/              how it gets built. The LAST document before code.
├── runbooks/           steps only the owner can do — dashboards, DNS, secrets
├── OpenQuestions.md    decisions waiting on the owner
├── Deferred.md         agreed, explicitly parked, with a trigger to unpark
├── KnownGaps.md        built, but not proved
├── Bugs.md             wrong behaviour in the running app
└── Ideas.md            unscoped, may never be built
```

## Lifecycle

```
idea ─────► Ideas.md          a line or two. No procedure, no ceremony.
  │
  │ (owner picks it up)
  ▼
spec ─────► specs/<date>-<slug>.md
  │         User stories, acceptance scenarios, what the interface should
  │         feel like. NO technology — no tables, endpoints, or frameworks.
  │
  │ (owner reviews — the cheapest point to catch a wrong direction)
  ▼
plan ─────► plans/<date>-<slug>.md
  │         The technical how, detailed enough to build straight from.
  │
  │ (owner approves)
  ▼
code
```

**Three stages, not five.** There is no task folder and no queue. The plan is the executable
artifact: a checklist of concrete steps, the files each touches, and how each is verified. An
earlier attempt at this project decomposed every plan into task documents and spent its budget
writing them instead of shipping. That layer existed because coding agents couldn't hold a plan and
exercise judgment at once — not the constraint any more.

**Internal work skips the spec.** Anything that only changes how the repo is built, checked, or
documented goes straight to a plan whose Spec row reads `n/a (internal)`. Anything the owner can
see, use, or reach starts with a spec.

## The rule that matters

**A plan must be buildable without asking the owner anything.** Every decision it needs is made in
it. That is the whole reason the spec comes first and gets reviewed: uncertainty is resolved
there, so the plan can be certain.

### One blocked piece doesn't stop the work

If writing a plan surfaces a genuine question, it goes to `OpenQuestions.md` and **only the
checklist item that depends on it waits**:

```markdown
- [x] Stream daemon events into the message store
- [~] Auto-title the conversation from its first message   ← blocked → OQ-1
- [x] Replay buffer, oldest sequence first
```

Everything else still gets built. The work is **done except OQ-1** — a real, closeable state, not
a stalled one. When the question is answered: unblock the item, finish it, delete the entry.

### Shipping without proof is allowed. Shipping without *saying so* is not

Verification sometimes can't be completed — no deployment yet, the platform won't emit the signal,
the surface doesn't exist. That's normal, and not a reason to hold a change back.

It **is** a reason to write it down. Whenever something is ticked on weaker evidence than it asked
for: say what you actually ran, and open a [`KnownGaps.md`](KnownGaps.md) entry in the same change.

A ticked box that quietly means "looked right to me" devalues every other ticked box in the repo,
and the next session has no way to tell which is which. A caveat that lives only in a chat message
does not exist.

## Which file does this go in?

| Situation | File |
|---|---|
| "Let's do that later" | `Deferred.md` |
| "I'm not answering that right now" | `OpenQuestions.md` |
| "It's built, but I couldn't prove it works" | `KnownGaps.md` |
| "It works, but only within these limits" | `KnownGaps.md` |
| "Might be nice one day" | `Ideas.md` |
| "This is behaving wrong" | `Bugs.md` |
| "Here's how I want to use it, and what it should feel like" | `specs/` |
| "Here's exactly how we build it" | `plans/` |
| "Only a human can do this part — dashboard, DNS, secrets" | `runbooks/` |

Each register states its own format at the top. Only specs, plans, and runbooks have templates,
because only those are long enough to need one.

## Open questions carry options

Per [`AGENTS.md` §4](../AGENTS.md), an entry in `OpenQuestions.md` needs context, a plain
user-side scenario, and concrete options — each with its own context, **the question's scenario
replayed under that option**, pros and cons, a score out of 10, blast radius if chosen wrong,
caveats, and a recommendation.

A question with no options is not ready to be asked. Options describing *different* situations
from one another cannot be compared, and aren't ready either.

Most decisions should never reach this file. The owner's standing preference is that agents
recommend and he vetoes — reserve it for choices that are genuinely his.
