---
name: writing-plans
description: >-
  Step-by-step procedure for authoring or revising a docs/plans/*.md technical
  plan from an owner-reviewed spec, per docs/templates/plan.md:
  foundational-vs-per-story work breakdown, Decisions with rejected
  alternatives, contract/data-model delegation, and mapping the spec's
  SC-nnn criteria to Verification. Use whenever writing, revising, or
  reviewing a plan.
---

# Writing a docs/plans/*.md plan

Read `docs/templates/plan.md` and `.sparstrowgen/blueprint.yaml` (stack and
commands) before writing anything. The template's structure — header table
(Spec/Status/Depends on/Touches/Tasks/Open questions), a foundational-vs-
per-story Work breakdown, Decisions with rejected alternatives, Phases,
Scope boundaries, and Verification mapped to the spec's `SC-nnn` criteria —
is this repo's actual planning discipline, not a suggestion.

## Read the spec fully first

Including its Assumptions and any `[NEEDS CLARIFICATION]` markers still
open. An open `OQ-n` the spec references blocks only the part of the plan
that depends on it — plan around it per `AGENTS.md` §4's options framework,
don't stall the whole plan for one unresolved thread.

## Splitting the work

Use the plan template's own test: **can the owner see the result?** Yes →
per-story, grouped so each story's phase ends in something demoable. No →
foundational (schema, protocol, transport, migrations) — it blocks the story work
behind it. A Work breakdown with stories and no rows under them is the exact
failure `docs/tasks/README.md` warns about: everything called foundational,
no story ever ships. Don't let that happen here.

## Decisions

For every load-bearing technical choice, write it under Decisions: the
choice, the alternative(s) rejected, and why. Six months from now the code
shows what was built; this section is the only place that shows why the
alternatives lost. Don't skip an entry because the choice felt obvious in
the moment.

## Contracts and data model — delegate, don't re-derive

A shared shape crossing the Go/TypeScript boundary — server to daemon, or
server to web — is defined once in `proto/` and generated into both
languages. Never describe such a shape in prose in a plan and let each side
implement it separately; that is the drift the generated types exist to
prevent. Write the message and its fields into the plan's Decisions section,
and let the task that lands it edit `proto/`.

Database changes are the same discipline in the other direction: the plan
names the tables, columns, and indexes it needs and why, and the task writes
the migration under `server/migrations/`. Schema is a foundational-work
decision, so it is settled in the plan rather than improvised inside a task.

## Verification

Map every one of the spec's `SC-nnn` success criteria to a concrete check
under Verification. If part of it can't be verified yet (no deployment, no
second machine, the platform won't deliver the signal), say so here — that
is what `docs/KnownGaps.md` is for, named early rather than discovered at the
end.

## Closing out

Leave `Status` as `Draft` until the plan is actually approved — the same
owner-gate discipline as the spec it came from. Never restate the spec's
reasoning in the plan — link it; one copy of "why," not two that drift
apart.
