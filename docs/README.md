# docs/

Everything that isn't code but needs to survive a session. Six files and four folders — if you're
unsure where something goes, it's one of these.

```
docs/
├── Capabilities.md    what the backend can deliver — READ BEFORE DESIGNING
├── Decisions.md       load-bearing choices, and what they beat
├── KnownGaps.md       built-but-unproved, and caveats noticed in passing
├── Unverified.md      checks not yet run on built things — verify them when time permits
├── Bugs.md            wrong behaviour in the running app
├── Later.md           questions, parked work, and ideas — one file, one format
├── feedback/          what he said after looking at a built surface, verbatim
├── specs/             what the owner wants and why, in his words
├── runbooks/          steps only he can do — dashboards, DNS, secrets
└── templates/         spec, runbook
```

The proposed development, staging, production and installed-daemon flow is
[`runbooks/release-workflow.md`](runbooks/release-workflow.md). Its matching agent protocol is the
repository-root [`WORKFLOW.md`](../WORKFLOW.md); the **Active phase** there decides which branch
rules apply.

There is **no plan document**. See [`AGENTS.md` §2](../AGENTS.md): a spec says what he wants, a
rendered design answers what it looks like, and that design's handoff contract is the backend's
brief.

## Where a feature actually lives

Mostly not here:

```
docs/specs/<date>-<slug>.md                         what he asked for, approved
docs/design/prototypes/<Category>/<name>.dc.html     the locked design
docs/design/prototypes/<Category>/<name>.handoff.md  what the backend must provide
docs/design/shots/<date>-<slug>/                     image directions and why one was chosen
proto/                                              shapes crossing Go ↔ TypeScript
server/migrations/                                  the schema it needed
```

## Which file?

| Situation | File |
|---|---|
| "Can the backend actually do this?" | `Capabilities.md` — and if it isn't answered there, answer it there |
| "We chose X over Y, here's why" | `Decisions.md` |
| "Built, but I couldn't prove it" / "works only within these limits" | `KnownGaps.md` (`unproved`) |
| "Here is a check nobody has run yet" | `Unverified.md` — one entry per check; verify them when time permits |
| "I noticed something fragile and left it alone" | `KnownGaps.md` (`caveat`) |
| "This is behaving wrong" | `Bugs.md` |
| "Later" / "just an idea" / "I'm not answering that now" | `Later.md` |
| "Here's what I want and how I'd use it" | `specs/` |
| Several changes at once, after looking at something built | `feedback/` — capture verbatim, triage after |
| "Only a human can do this part" | `runbooks/` |
| "Which branch, environment or release channel does this use?" | `../WORKFLOW.md` |

Each file states its own format at the top. Only specs and runbooks have templates, because only
they're long enough to need one.

## The three rules that make this work

**A spec never describes an interface.** "A sidebar showing recent conversations" is a design
decision smuggled into prose, and it pre-empts the options he's meant to choose between. Say what
someone needs to do and what's true afterwards.

**One blocked piece doesn't stop the work.** A real question goes to `Later.md` as a `question` and
blocks only the thing depending on it. Build the rest and report **done except L-n** — a closeable
state, not a stalled one.

**Shipping without proof is allowed. Shipping without *saying so* is not.** Sometimes a check can't
be run — no deployment, no signal, the surface doesn't exist. That's normal and not a reason to
hold a change. It *is* a reason to say what you actually ran and open a `KnownGaps.md` entry in the
same change. A caveat that lives only in a chat message does not exist to the next session.
