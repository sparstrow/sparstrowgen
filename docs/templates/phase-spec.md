<!--
TEMPLATE — copy to docs/tasks/<phase>/README.md, then delete every HTML
comment in the copy.

A phase spec holds what ALL of a phase's tasks share, so a decision is written
once and referenced — not copy-pasted into eight files and then updated in six.
If exactly one task needs a decision, it belongs in that task, not here.

Write this BEFORE the individual tasks. Decomposing a phase is where the real
design happens: every phase in this repo so far has surfaced load-bearing
decisions that were invisible from the plan's bullet list.
-->

# <Phase id> — <name>

| | |
|---|---|
| **Plan** | <docs/plans/<file>.md (<phase id>)> |
| **Kind** | <**foundational** — blocks stories, demos to nobody \| **serves <US-n>** — ends in something the owner can use> |
| **Spec** | <docs/specs/<file>.md, or "n/a (internal)"> |
| **Depends on** | <phases that must land first, or —> |
| **Blocks** | <phases waiting on this, or "nothing"> |
| **Status** | <not started \| NN–NN done <date> \| ✅ done <date>> |
| **Open questions** | <none \| OQ-n, blocking only task NN> |

<!--
KIND decides how this phase's tasks are shaped, per its plan's Work breakdown:

  foundational  → ordinary technical tasks (schema, transport, sync). Nothing
                  here demos. Its job is to unblock the story phases behind it.
  serves US-n   → tasks grouped so the phase ENDS in a working, demoable
                  surface. Its Definition of done is the story's acceptance
                  scenarios, not a list of components built.

A phase that claims to serve a story but whose Definition of done contains no
scenario the owner could walk through is mislabelled — it is foundational.
-->

## The story this serves

<!--
DELETE THIS SECTION for a foundational phase.

Quote the user story from the spec — the journey, its acceptance scenarios, and
its independent test — so the tasks are graded against the owner's words rather
than a paraphrase of them. Cite US-n; do not restate the whole spec.
-->

> **<US-n> — <title>** (<docs/specs/<file>.md>)
>
> <the journey, quoted>

**Acceptance scenarios this phase must satisfy:**

1. **Given** <…>, **When** <…>, **Then** <…>

**Independent test:** <from the spec — what proves this story alone works>

## The four states

<!--
DELETE THIS SECTION for a foundational phase.

Every surface this phase ships needs all four, per the spec's Interface &
experience section and AGENTS.md §3.9. Name them here once so no task has to
guess and the verification task has something concrete to check.

The empty state is the one that gets skipped and the one the owner sees first.
-->

| Surface | Populated | Empty | Loading | Error |
|---|---|---|---|---|
| <surface> | <…> | <what it says + the action it offers> | <skeleton> | <what failed + next action> |

## Tasks

Run order and concurrency live in [`../MasterTaskQueue.md`](../MasterTaskQueue.md).

| Task | Tag | Serves | Depends on | Status |
|---|---|---|---|---|
| [<T-id> — <name>](<file>.md) | `<[S]\|[P]\|[C]>` | <US-n \| foundational> | <— or task ids> | <not started> |

<!--
Tags — exactly one per task, defined in ../README.md:
  [S] Sequential — blocks dependents, run alone
  [P] Parallel   — no shared files with siblings, hand to different agents
  [C] Concurrent — any order, but touches shared files, one worker at a time

The practical test for [P] vs [C]: could two agents start these RIGHT NOW
with zero coordination? If they'd collide on a file, it's [C].

Every phase should end with a verification task tagged [S] — see
verification-task.md.
-->

This file holds what they share. Individual tasks reference it rather than
restating it.

## Objective

<!--
What this phase delivers, in a few sentences. Not the plan's framing repeated —
the concrete shape of the work now that someone has read the code.
-->

## The shape of what was found

<!--
DELETE if decomposition turned up nothing surprising — but check honestly
first, because it usually does.

This section is for things established by READING THE CODE that change the
work from what the plan assumed: a premise that stopped being true, a piece
already built, an adapter that solves half the problem, a route name the plan
got loose about.

This is the highest-value section in a phase spec. It is where a plan's loose
assumption gets caught — a step that assumed a deployment nobody had made, a
dead WebSocket, an unpaginated transcript. All of those would
have been discovered anyway — halfway through implementation, at much higher
cost.
-->

## Definition of done

<!--
The observable outcomes, as a list. Each one either happened or didn't — no
"improved", no "better". If a reader can argue about whether an item is met,
rewrite it.

FOR A STORY PHASE, the first items are the story's acceptance scenarios,
walked end to end, plus all four states on every surface it ships. A story
phase that is "done" while its empty state is a bare "No items" is not done —
see AGENTS.md §3.9 and the spec's Interface & experience section.

FOR A FOUNDATIONAL PHASE, the outcomes are technical and that is correct. Say
which story phase it unblocks, so the thing it exists to serve is named.

End with what is explicitly NOT in this phase, pointing at the decision that
excluded it. That sentence is what stops the next agent scope-creeping.
-->

- <observable outcome — for a story phase, an acceptance scenario walked>
- `pnpm typecheck` and `pnpm test` stay green

**Not in this phase:** <what, and which decision says so>

---

## Decisions already made

<!--
The shared decisions, each under its own ### heading with a claim as the
title — "The five routes are thin re-exports. Resist making them anything
else" beats "Routing approach".

Include the reasoning and the rejected alternative. A decision without its
"instead of what" gets re-litigated by the next agent who has a different
instinct.

Decisions INHERITED from the plan get cited, not restated. Only decisions made
here, during decomposition, get written out in full.
-->

### 1. <claim as the heading>

<!-- Reasoning, and what was rejected. -->

---

## The owner action this phase cannot do for itself

<!--
DELETE if there isn't one.

For work that is fully decided but needs a human: an account, a dashboard
setting, a secret. This is NOT an open question — nothing is undecided,
someone just has to go do it.

It needs a matching row in ../../runbooks/README.md, which is where the owner
actually goes to act on it. This section explains why the phase is exposed to
it; that file is the checklist.
-->

## Files

| Path | Change |
|---|---|
| `<path>` | <new \| edit — what changes> |

## Traps

<!--
Failure modes that would be hit otherwise, each with WHY it bites and what to
do instead. Bold the claim, explain underneath.

The traps worth writing are the ones that fail QUIETLY — a param that arrives
undefined and renders an empty state, a route that compiles and is linked from
nowhere. A loud failure teaches itself; a silent one gets shipped.
-->

## Verification

<!--
The assertions that matter, numbered. The full procedure lives in the
verification task; this is what it is graded against, stated once so the tasks
can point at it.
-->

Full procedure in [<T-id> — verification](<file>.md).
