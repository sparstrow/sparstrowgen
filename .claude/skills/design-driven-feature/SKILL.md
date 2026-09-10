---
name: design-driven-feature
description: >-
  The workflow for building any user-facing feature in this repo: check what
  the backend can deliver, show the owner two or three real rendered design
  directions, wire the chosen one into the actual app on placeholder data, get
  it confirmed, and only then build the backend to serve exactly that design.
  Use this for every feature request, every "can we add", every "I want a
  screen that", and any time work is about to start on something the owner
  will see. A spec captures what the owner wants in his words and is approved
  before designing; there is no plan document — the design answers what it
  looks like.
---

# Design-driven feature work

A previous attempt at this app was plan-driven: the planning took the time, the
coding and testing didn't, and it never shipped. So the order is inverted, and
the number of documents is small and fixed.

**Two documents, each with exactly one author.** The owner writes the spec —
what he wants and why, in his terms, as scenarios he can elaborate. The design
answers what it looks like, and he picks it from rendered options rather than
reading a description of it. Nothing else is written before code: no plan, no
tasks.

Anything visual is decided by looking at it, never by prose.

## The loop

```
Spec         what he wants and why, in his words. He approves it.
Feasibility  can the backend actually deliver it?
Shots        2–4 directions as generated IMAGES. He picks a direction. ~2 min each.
Design       the chosen direction, built as a clickable prototype. He picks details.
Wire         into the real app on mock data. He confirms it there.
Backend      built to the locked design's contract. Mocks swapped out.
Verify       frontend-verify, against real data.
```

Everything up to Wire is cheap and fast. **Backend is the expensive phase and
it does not start until he has confirmed the design in the app.** That ordering
is the whole point.

**Shots come before Design for the same reason Design comes before Backend:**
each step is cheaper than the one after it, so the expensive step only ever runs
on something already wanted. Coding a prototype caps exploration at two
directions in practice; images make four affordable, so the prototype gets built
on a direction he chose rather than the first one anybody thought of.

## Spec — when there is something to explain

The owner's statement of what he wants and why, as user scenarios. `writing-specs`
carries the procedure; the short version is **draft it from what he already
said and hand it back for correction** rather than interviewing him for it.

Two things a spec must never contain: technology, and interface design. The
second is the one that gets broken — "a sidebar showing recent conversations"
is a design decision in prose, and it pre-empts the options he is supposed to
choose between in the Design phase.

Nothing designs against a Draft. `Status: Approved <date>` first.

**Skip this step** for bug fixes, backend-only work, and small specific
changes. A spec earns its place when he has something to explain, not as a
formality.

## Feasibility — before drawing anything

**Read [`docs/Capabilities.md`](../../../docs/Capabilities.md) before designing
anything.** It says what the backend can and cannot produce.

Designing something the backend cannot achieve is pure waste: the screen looks
finished, the owner approves it, and then it can't be served. This is the
failure this workflow exists to prevent, and it is the one the owner named
directly.

If the design needs something not in `Capabilities.md`, pick one — never
"probably fine":

- **Check it.** Run the CLI, capture the output, add a verified row. Usually
  minutes.
- **Design around it.** Change the design to need only what's deliverable.
- **Cut it.** Record it in `Later.md` with a trigger and ship the design
  without that piece.

## Shots — pick a direction from pictures, in minutes

**Invoke `design-shotgun`.** It generates 2–4 directions as images via codex,
shows them side by side, and records why the loser lost.

An image settles composition, hierarchy, density and mood. It settles nothing
about interaction, real data, or the four states — so say that when presenting,
or the next step feels like re-opening a closed question. Images pick a
direction; the prototype makes the decision.

Skip it for small specific changes, for one more screen in an established
pattern, and when codex is unavailable — in that last case go straight to two
prototype directions rather than describing images in prose.

## Design — show, don't describe

The chosen direction, made clickable. Where a shot is ambiguous — and it will be,
because images have no states — build the ambiguity out rather than resolving it
silently.

If Shots was skipped, this step carries the exploration instead: two or three
directions, **genuinely different** — different layout, different information
hierarchy, different interaction model. Three variations on one idea with
different spacing is one option, not three.

Two ways to render them, both valid:

- **Standalone prototype** — `interactive-prototype` builds it into
  `design-system/designs/`. Right when the app route doesn't exist yet, or when
  comparing directions side by side.
- **Placeholder in the real app** — a real route in `apps/web`, real
  components, mock data. Right when the surrounding app exists, because
  context changes judgment. Prefer this once there is an app to put it in.

Build against `DESIGN.md` and the design system's tokens, with
`ai-design-slop` loaded so the tells never go in. A prototype that invents its
own visual language teaches the owner nothing about what the real thing will
look like.

**Don't run the "Presenting a choice" framework (`AGENTS.md` §4) on anything
you can render.**
That framework is for decisions with no picture — a protocol choice, a
tradeoff between two libraries. For design, seeing them *is* the comparison.
Say what each direction is optimising for in a line, then let them look.

## Wire — lock it, wire it, confirm it

The owner picks a direction, or asks for pieces of two. Fine — merge and
re-show rather than arguing for one.

Once locked, **wire it into the real app on placeholder data.** Real route,
real components, real states. This is the step that catches what a mockup
can't: how it feels next to the rest of the app, how it behaves at real
viewport sizes, whether the empty state is the one they'll actually see most.

Placeholder data conventions:

- Mock modules are named `*.mock.ts` and live beside what they feed.
- **A feature is not done while a shipped route still imports a `*.mock.ts`.**
  That makes leftovers greppable rather than a thing to remember.
- Mock data looks realistic, not `foo` / `bar`. Fake data that is too tidy
  hides the layout problems real data will cause — long names, empty fields,
  one item, two hundred items.

All four states are present before the owner confirms (`AGENTS.md` §4, rule 5).
Judging a design on its populated state alone is how empty states end up
designed by accident.

## Backend — the contract falls out of the locked design

The design now says exactly what data it needs. Write that down as the
prototype's handoff contract — `interactive-prototype` produces one, and its
**Data contract** section is the backend's brief.

State, per surface: the fields, their types, whether they stream or arrive
whole, what "loading" and "empty" and "error" actually mean here, and what
must survive a refresh.

This is what a plan document used to be, except it is derived from a design
the owner has already approved — so it cannot describe a feature nobody asked
for. Tie it back to the spec's acceptance scenarios where they apply.

### Build it, then swap the mocks

Build to the contract. **Nothing speculative** — no field the design doesn't
show, no endpoint nothing calls. If building reveals the contract was wrong,
go back and say so; don't silently widen it.

Shapes crossing the Go/TypeScript boundary are defined once in `proto/` and
generated. Schema changes go to `server/migrations/`. Record the load-bearing
choices and their rejected alternatives in
[`docs/Decisions.md`](../../../docs/Decisions.md) — a few lines, not a
document.

Then delete the mocks and wire the real data. Grep for `*.mock.ts` to prove
none are left.

## Verify

`frontend-verify`, against the real app on real data. A green typecheck is not
evidence that a feature works.

## When this workflow does not apply

- **Backend-only work with no surface** — the daemon connection, a migration,
  a protocol change. Build it; there is nothing to spec or design.
- **A bug fix.** Fix it.
- **The owner asks for something specific and small.** "Make that button
  secondary" does not need three directions. Read the room: this workflow is
  for features, not for every edit.
