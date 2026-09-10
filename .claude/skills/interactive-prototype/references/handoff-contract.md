# The handoff contract

Every prototype ships a sibling `<name>.handoff.md`. It is the document whoever
builds the real thing reads **instead of** reverse-engineering the prototype's
markup.

That distinction is the whole reason this file exists. A prototype is a
throwaway artifact built for speed: it carries scaffolding, shortcuts, and
placeholder decisions that must *not* be lifted into production. Handing over
the raw prototype invites someone to port its structure faithfully, including
the parts that were never meant to survive. **The handoff is the contract; the
prototype is the reference.** When they disagree, the handoff wins.

## Template

```markdown
# <Surface name> — handoff

| | |
|---|---|
| **Prototype** | `<name>.dc.html` |
| **Provenance** | `docs/specs/<file>.md` — or `exploratory — no spec` |
| **Mode** | build \| explore |
| **Status** | draft \| reviewed <date> \| superseded by <x> |
| **Design system** | mirror \| greenfield, at `design-system/` |

## What this is

<Two sentences. What the surface does and who uses it.>

## Component mapping

Every element in the prototype, mapped to what should build it. This is what
stops a reimplementation of something that already exists.

| Prototype element | Use | Notes |
|---|---|---|
| Status pill | existing `Badge` | `variant="ok"` for Shipped |
| Row drawer | existing `Sheet` | |
| Bin matrix | **NEW — nothing covers this** | See "New components" below |

## Token usage

Tokens consumed, and — critically — any the prototype needed that the system
does not have. A missing token is a design-system decision, not something the
implementer should invent.

| Needed | Exists? | Action |
|---|---|---|
| `--warn-bg` | yes | — |
| `--table-row-hover` | **no** | Add to the system, or reuse `--muted` |

## States

| State | Reachable in prototype | Notes |
|---|---|---|
| Populated | yes | 12 rows |
| Empty | yes — `?state=empty` | Copy is final |
| Loading | yes | Skeleton matches row shape |
| Error | yes — `?state=error` | Copy is placeholder |

## Data contract

What the surface needs, in plain terms — fields, shapes, and where each comes
from. Flag anything with no backend behind it today; that is scope the plan has
to account for.

| Field | Source | Exists? |
|---|---|---|
| `order.total` | orders API | yes |
| `order.riskScore` | — | **no backend — new work** |

## Interactions

The behaviours that are load-bearing, and what is faked.

- Status tabs filter client-side — real behaviour, keep.
- "Advance status" mutates local state only — **needs a real endpoint**.
- Export button is inert — decoration, do not port.

## Invented

Everything here was decided by the prototype and approved by nobody. This is the
most important list in the document — the shortest section people skip and the
one that causes rework.

- Empty-state copy.
- The three-column split at ≥1440px.
- Sorting defaults to order date descending.

## Open questions

Anything the spec left open that the prototype had to render somehow. Say which
way it was rendered and that the choice is not a decision.

- OQ: does an invoiced order stay in the default list? Prototype shows it does.

## Not included

Deliberate omissions, so nobody reads absence as oversight.

- Bulk actions — deferred, see `docs/Deferred.md`.
- Print view — out of scope.
```

## Filling it in well

**"Invented" is the section that earns this document.** Every prototype decides
things nobody asked for — a default sort, a breakpoint, a piece of copy. Those
decisions get built and shipped as though they were requirements unless someone
writes them down as guesses. Be exhaustive here even when it feels pedantic;
this is the list a reviewer scans to catch scope that arrived by accident.

**Be specific about what is faked.** "The export button is inert" prevents an
implementer spending a day wiring an export nobody specified.

**Name missing backend work explicitly.** A prototype makes new data look free
because seed data is free. `riskScore` rendering beautifully in HTML says
nothing about whether anything can compute it.

**Do not restate the spec.** Link it. The handoff covers what the *prototype*
adds, assumes, or leaves open — one copy of the requirements, not two that drift.

## Where it goes

Next to the prototype: `design-system/designs/<Category>/<name>.handoff.md`.

In a repo running a spec → plan → tasks lifecycle, the handoff is an input to
the **plan**, not a replacement for it. The spec says what the owner wants, the
prototype shows it, the handoff records what building it will actually involve —
and the plan is still where the technical decisions get made and recorded.
