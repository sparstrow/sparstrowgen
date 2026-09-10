---
name: writing-specs
description: >-
  Step-by-step procedure for authoring or revising a docs/specs/*.md
  specification per docs/specs/README.md and docs/templates/spec.md: eliciting
  the WHAT and WHY, prioritized P1/P2/P3 user stories, Given/When/Then
  acceptance scenarios, all four interface states, and handling open
  questions. Use whenever writing, revising, or reviewing a spec.
---

# Writing a docs/specs/*.md spec

Read `docs/specs/README.md` and `docs/templates/spec.md` before writing
anything — their rules are mandatory, not house style: no technology, every
story independently demoable, all four interface states,
`[NEEDS CLARIFICATION: …]` markers for genuine unknowns.

## Before writing: is this new ground or a revision?

- **New feature** → check `docs/Ideas.md` (is this already a captured idea to
  promote, with its context intact?) and `docs/specs/README.md`'s index (does
  a spec already exist to extend instead of duplicating?).
- **Existing surface changing shape** → read the current spec if any, plus
  `docs/KnownGaps.md` and `docs/Deferred.md` for that surface, before writing
  "The experience today" — that section is a factual account of the app as
  it stands, not a paraphrase of the request.

## The elicitation

Elicit the WHAT and WHY; resist the HOW. If a sentence names a table,
endpoint, component, or framework, it belongs in the plan — cut it.

Prioritize stories P1/P2/P3; each must stand alone (buildable, testable,
demoable by itself). A "P2" that only makes sense bundled with P1 is a
technical step wearing a story's clothes — push it into the plan's
foundational work instead and say so under Assumptions, not silently drop
it.

Write Given/When/Then acceptance scenarios per story, including at least one
unhappy-path scenario — the failure path is what decides whether a feature
feels trustworthy, not the happy path everyone builds anyway.

## Interface & experience

Name every surface (new vs. existing, one line on what it's for), and give
all four states:

| State | What to specify |
|---|---|
| **Populated** | the normal case |
| **Empty** | what it says, and the action it offers — never a bare "No items" |
| **Loading** | skeleton shaped like the real content, or say what else |
| **Error** | what failed, in plain words, and the next action |

A surface with only the populated state described is not finished; the
empty state is what the owner sees on day one, and the one most often
skipped.

## Requirements and success criteria

Write numbered functional requirements (`FR-00n`, testable against a running
app — never implementation), key entities (concepts, not schema), and
measurable success criteria (`SC-00n`).

## Handling unknowns

Mark genuine unknowns inline: `[NEEDS CLARIFICATION: <what's unknown>]`. If
an unknown actually blocks *writing* the spec (not just building it),
promote it to an `OQ-n` entry in `docs/OpenQuestions.md` using the full
`AGENTS.md` §4 options framework — pros/cons, score /10, blast radius,
caveats, recommendation. Only that thread blocks; keep writing the rest of
the spec around it.

## Closing out

1. Record scope boundaries explicitly under Assumptions: what this
   deliberately excludes, and where it went — an existing `Deferred.md` /
   `Ideas.md` id, or a new one filed in the same turn per `AGENTS.md`'s
   "document it the turn it surfaces" rule.
2. Add or update the spec's row in `docs/specs/README.md`'s index table.
3. Leave `Status` as `Draft` and `Owner review` empty — never fill in your
   own review. Planning does not start on an unreviewed spec.

## Working style

Spec-writing is normally a back-and-forth with whoever wants the feature,
not a fire-and-forget brief — when working interactively, ask clarifying
questions in conversation as you go. When working from a static delegated
brief instead, with no live back-and-forth available, use inline
`[NEEDS CLARIFICATION]` markers and `OpenQuestions.md` entries, and say so
explicitly in the final summary so whoever delegated the work knows what's
still open.
