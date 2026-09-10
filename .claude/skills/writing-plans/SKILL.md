---
name: writing-plans
description: >-
  Step-by-step procedure for authoring or revising a docs/plans/*.md plan from
  an owner-reviewed spec, per docs/templates/plan.md. The plan is the LAST
  document before code — it must be executable as written, with no task-file
  layer beneath it. Covers the work breakdown, Decisions with rejected
  alternatives, naming the files and protocol/schema shapes up front, and
  mapping the spec's SC-nnn criteria to Verification. Use whenever writing,
  revising, or reviewing a plan.
---

# Writing a docs/plans/*.md plan

Read `docs/templates/plan.md` and `.sparstrowgen/blueprint.yaml` (stack and
commands) before writing anything.

**The plan is the last document before code.** There is no task folder, no
queue, no per-task file. A previous attempt at this project decomposed every
plan into task documents and spent its budget writing them instead of
shipping; that layer existed because coding agents could not hold a plan and
exercise judgment at the same time, which is no longer the constraint.

So the bar for a plan is different from the usual one: **an agent should be
able to open it and build, without asking the owner anything.** Every decision
it needs is made here.

## Read the spec fully first

Including its Assumptions and any `[NEEDS CLARIFICATION]` markers still open.
An open `OQ-n` blocks only the part of the plan that depends on it — plan
around it per `AGENTS.md` §4's options framework, don't stall the whole plan
for one unresolved thread.

## Splitting the work

Use the plan template's own test: **can the owner see the result?** Yes →
per-story, grouped so each group ends in something demoable. No →
foundational (schema, protocol, transport, migrations) — it blocks the story
work behind it.

A work breakdown that is all foundational and no stories is the failure to
watch for: every layer gets built, each passes its tests, and the thing the
owner wanted to use never arrives.

Keep the whole plan to something that can actually land. If it is too big to
finish, it is too big to be one plan — split it and write the second one
later, against the shipped shape of the first.

## Make it executable

This is the part that replaces the task layer. For each unit of work:

- **A checklist of concrete steps**, each one tickable. Not "implement the
  daemon connection" — the actual steps, in order.
- **The files it touches**, named. If a file doesn't exist yet, say where it
  goes.
- **How it is verified.** The command to run, or the thing to click. Tie it to
  the spec's `SC-nnn` where one applies.
- **What is explicitly out of scope**, when the boundary is not obvious.

Zero open questions may remain inside a unit of work. If writing one surfaces
a genuine question, it goes to `docs/OpenQuestions.md` and blocks **only** the
checklist item that depends on it — mark that item `[~] blocked → OQ-n` and
leave the rest buildable. "Done except OQ-n" is a real, reportable state.

## Decisions

For every load-bearing technical choice: the choice, the alternatives
rejected, and why. Six months from now the code shows what was built; this
section is the only place that shows why the alternatives lost. Don't skip an
entry because the choice felt obvious at the time.

## Protocol and schema shapes belong here

A shape crossing the Go/TypeScript boundary — server to daemon, or server to
web — is defined once in `proto/` and generated into both languages. Never
describe such a shape in prose and let each side implement it separately; that
is the drift the generated types exist to prevent. **Write the actual message
and its fields into Decisions**, so the implementing step is transcription
rather than invention.

Database changes work the same way: the plan names the tables, columns, and
indexes and why, and the implementing step writes the migration under
`server/migrations/`. Schema is foundational, so it is settled here, never
improvised mid-build.

## Verification

Map every one of the spec's `SC-nnn` criteria to a concrete check. If part of
it can't be verified yet — no deployment, no second machine, the platform
won't deliver the signal — say so here rather than discovering it at the end,
and open the `docs/KnownGaps.md` entry when it lands that way.

Anything a browser can exercise is verified in a browser, per `AGENTS.md`
§3.8. A green typecheck is not evidence that a feature works.

## Closing out

Leave `Status` as `Draft` until the plan is actually approved — the same
owner-gate discipline as the spec it came from. Never restate the spec's
reasoning; link it. One copy of "why", not two that drift apart.
