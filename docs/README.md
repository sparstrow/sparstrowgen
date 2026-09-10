# docs/

Everything that isn't code but needs to survive a session. There are no plan documents — see
[`AGENTS.md` §1](../AGENTS.md). A spec says what the owner wants; the design answers what it
looks like.

```
docs/
├── specs/             what the owner wants and why, in his words. No technology, no layouts.
├── Capabilities.md    what the backend can deliver — READ BEFORE DESIGNING
├── Decisions.md       load-bearing choices, and what they beat
├── Bugs.md            wrong behaviour in the running app
├── KnownGaps.md       built, but not proved
├── Deferred.md        agreed, explicitly parked, with a trigger to unpark
├── OpenQuestions.md   decisions waiting on the owner
├── Ideas.md           unscoped, may never be built
├── runbooks/          steps only the owner can do — dashboards, DNS, secrets
└── templates/         skeletons — spec, runbook
```

## Where a feature actually lives

Not here. A feature's record is the design the owner approved and the code that serves it:

```
docs/specs/<date>-<slug>.md                         what the owner asked for, approved
design-system/designs/<Category>/<name>.dc.html     the locked design
design-system/designs/<Category>/<name>.handoff.md  what the backend must provide
proto/                                              the shapes crossing Go ↔ TypeScript
server/migrations/                                  the schema it needed
```

The handoff contract is what a plan document used to be, except it is derived from something the
owner already looked at and approved — so it cannot describe a feature nobody asked for.

## The two files that get read most

**[`Capabilities.md`](Capabilities.md)** — what the backend can and cannot produce. Read it before
designing anything. Designing something undeliverable is the specific waste this project is
organised to avoid: the screen looks finished, it gets approved, and then it can't be served.

**[`Decisions.md`](Decisions.md)** — why each load-bearing choice beat its alternatives. Appended
to when a choice would be expensive to reverse; skipped when it's cheap to change later. A few
lines, never a document.

## Which file does this go in?

| Situation | File |
|---|---|
| "Can the backend actually do this?" | `Capabilities.md` — and if it isn't answered there, answer it there |
| "We chose X over Y, and here's why" | `Decisions.md` |
| "This is behaving wrong" | `Bugs.md` |
| "It's built, but I couldn't prove it works" | `KnownGaps.md` |
| "It works, but only within these limits" | `KnownGaps.md` |
| "Let's do that later" | `Deferred.md` |
| "I'm not answering that right now" | `OpenQuestions.md` |
| "Might be nice one day" | `Ideas.md` |
| "Here's what I want and why, and how I'd use it" | `specs/` |
| "Only a human can do this — dashboard, DNS, secrets" | `runbooks/` |

Each register states its own format at the top. Only specs and runbooks have templates, because
only they are long enough to need one.

**A spec never describes an interface.** "A sidebar showing recent conversations" is a design
decision smuggled into prose, and it pre-empts the options the owner is meant to choose between.
Say what someone needs to do and what is true afterwards; the design answers the rest.

## Two rules worth repeating

**One blocked piece doesn't stop the work.** A genuine open question goes to `OpenQuestions.md`
and blocks only the thing that depends on it. Everything else still gets built, and the work is
reported **done except OQ-n** — a real, closeable state, not a stalled one.

**Shipping without proof is allowed. Shipping without *saying so* is not.** Verification sometimes
can't be completed — no deployment yet, the platform won't emit the signal, the surface doesn't
exist. That's normal and not a reason to hold a change back. It *is* a reason to say what you
actually ran and open a [`KnownGaps.md`](KnownGaps.md) entry in the same change. A caveat that
lives only in a chat message does not exist to the next session.

## Open questions carry options

Per [`AGENTS.md` §5](../AGENTS.md): context, a plain user-side scenario, and concrete options —
each with the scenario replayed under it, pros and cons, a score, blast radius, caveats, and a
recommendation.

**If it can be rendered, it doesn't belong here** — show the owner the designs instead. This file
is for decisions with no picture, and most decisions shouldn't reach it at all: the standing
preference is that agents recommend and the owner vetoes.
