---
name: writing-specs
description: >-
  Procedure for writing a docs/specs/*.md spec — the owner's statement of what
  he wants and why, in user-scenario form, which he approves before any design
  work starts. Draft it from what he has already said rather than interviewing
  him for it. Use when a feature is requested and there is no spec, when an
  existing spec needs revising, or when the owner is elaborating on something
  he wants. Do NOT describe interfaces, layouts, or technology in a spec — the
  design step answers what it looks like.
---

# Writing a spec

The spec is the **owner's** document. It says what he wants and why, in his
terms, elaborated enough that nobody downstream has to guess. It is the input
to the design step and the cheapest place to catch a wrong direction — a wrong
spec propagates into designs, backend, and everything after it.

Read `docs/templates/spec.md` before writing. It carries the shape; this file
carries how to fill it.

## Draft first, ask second

**Do not interview him for a spec he has effectively already given you.** He
is a business systems analyst — describing scenarios is his native mode, and
he usually supplies them in the request itself.

So: write the draft from what he said, mark what you inferred, and hand it
back for correction. Correcting a draft is faster and more accurate than
answering a list of questions, and it costs him one read instead of a session.

Ask only where the gap is genuine and you cannot responsibly guess — and ask
those questions *with the draft*, not before it.

## The two hard rules

**No technology.** No tables, endpoints, component names, frameworks, or file
paths. If a sentence couldn't be read aloud to someone who has never seen the
codebase, it belongs elsewhere.

**No interface design.** Do not describe layouts, screens, panels, or where
things sit. That is the design step's job, and prose describing a layout
pre-empts the options he is supposed to choose between — it quietly becomes
the decision. Say what someone needs to *do* and what should be *true*
afterwards; let the design answer what it looks like.

The second rule is the one that gets broken. Watch for "a sidebar showing…",
"a dropdown to pick…", "at the top of the screen" — each of those is a design
decision smuggled into a spec.

## What makes the stories good

- **Independently useful.** Build only that story and he still has something he
  can open and use. A story that delivers nothing alone is a technical step in
  a story's clothes.
- **Unhappy paths included.** Given/When/Then for what happens when it's empty,
  offline, too slow, or half-finished. This is where specs go thin, and it is
  exactly what the design's empty and error states get built from — a spec
  silent on them produces a design that invents them.
- **Two or three stories.** Ten means this is two specs.

## Capability claims don't go here

If the spec assumes the backend can produce something, that belongs in
[`docs/Capabilities.md`](../../../docs/Capabilities.md), verified, not in
Assumptions as a hope. The design step reads Capabilities before drawing —
which is what stops a spec from asking for something undeliverable and nobody
noticing until the backend is due.

## Closing out

- `Status: Draft` until he has actually read it and said yes. "No objections"
  is not approval; people don't object to documents they skimmed.
- Nothing designs against a Draft.
- Once approved, the design step takes over — see `AGENTS.md` §2. Fill the
  spec's **Design** row when a direction is locked, so the two stay linked.

## When to skip a spec entirely

Bug fixes. Backend-only work with no surface. Small specific changes ("make
that button secondary"). Anything where writing the spec takes longer than
doing the thing.

A spec earns its place when the owner has something to explain. It does not
earn its place as a formality.
