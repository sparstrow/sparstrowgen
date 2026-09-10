---
name: design-driven-feature
description: >-
  The workflow for building any user-facing feature in this repo: check what
  the backend can deliver, show the owner two or three real rendered design
  directions, wire the chosen one into the actual app on placeholder data, get
  it confirmed, and only then build the backend to serve exactly that design.
  Use this for every feature request, every "can we add", every "I want a
  screen that", and any time work is about to start on something the owner
  will see. Do NOT write a spec or plan document first — the design is the
  spec.
---

# Design-driven feature work

Nothing here starts with a document. It starts with something the owner can
look at.

A previous attempt at this app was plan-driven: the planning took the time,
the coding and testing didn't, and it never shipped. So the order is inverted.
The design is the specification — it communicates what the owner wants better
than prose ever did, and it is the thing they can actually judge.

## The loop

```
1  Feasibility   what can the backend actually deliver here?
2  Options       2–3 genuinely different directions, rendered
3  Owner picks   one direction, or a mix
4  Wire it       into the real app, on placeholder data
5  Confirm       owner uses it in the app, not a mockup
6  Contract      what the backend must provide, derived from the locked design
7  Backend       built to that contract, nothing speculative
8  Swap          placeholder data out, real data in
9  Verify        frontend-verify, against the real thing
```

Steps 1–5 are cheap and fast. Step 7 is the expensive one, and it does not
start until step 5 is done. That ordering is the whole point.

## 1 — Feasibility comes first, always

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
- **Cut it.** Record it in `Deferred.md` with a trigger and ship the design
  without that piece.

## 2 — Show, don't describe

Two or three directions, **genuinely different** — different layout, different
information hierarchy, different interaction model. Three variations on one
idea with different spacing is one option, not three.

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

**Don't run the AGENTS.md §5 options framework on anything you can render.**
That framework is for decisions with no picture — a protocol choice, a
tradeoff between two libraries. For design, seeing them *is* the comparison.
Say what each direction is optimising for in a line, then let them look.

## 3–5 — Lock it, wire it, confirm it

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

All four states are present before the owner confirms (`AGENTS.md` §4.9).
Judging a design on its populated state alone is how empty states end up
designed by accident.

## 6 — The contract falls out of the locked design

The design now says exactly what data it needs. Write that down as the
prototype's handoff contract — `interactive-prototype` produces one, and its
**Data contract** section is the backend's brief.

State, per surface: the fields, their types, whether they stream or arrive
whole, what "loading" and "empty" and "error" actually mean here, and what
must survive a refresh.

This replaces the plan document. It is derived from something the owner has
already approved, so it cannot describe a feature nobody asked for.

## 7–8 — Backend, then swap

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

## 9 — Verify

`frontend-verify`, against the real app on real data. A green typecheck is not
evidence that a feature works.

## When this workflow does not apply

- **Backend-only work with no surface** — the daemon connection, a migration,
  a protocol change. Build it; there is nothing to design.
- **A bug fix.** Fix it.
- **The owner asks for something specific and small.** "Make that button
  secondary" does not need three directions. Read the room: this workflow is
  for features, not for every edit.
