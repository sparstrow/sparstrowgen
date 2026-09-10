<!--
TEMPLATE — copy to docs/plans/<YYYY-MM-DD>-<slug>.md, then delete every
HTML comment in the copy.

A plan is the "how", derived from an owner-reviewed spec. The spec owns what
the owner wants and why; this document owns the technical shape that delivers
it, and MUST NOT restate the spec's reasoning — one copy, not two that drift.

  spec  → what the owner wants, in the owner's terms
  plan  → how it gets built, technically                  ← YOU ARE HERE
  tasks → the executable steps
  code

Uncertainty IS allowed here — that is why plans and tasks are separate files.
Anything still undecided goes to OpenQuestions.md with the AGENTS.md §4 options
framework, and the plan says so in its header rather than pretending the
question doesn't exist.

NO SPEC? Work that changes what the owner sees, does, or can reach needs one
first. Work that only changes how the repo is built, checked, documented, or
governed does not — those plans set Spec to "n/a (internal)" and say why in
one line. When it's genuinely unclear, ask.
-->

# <Plan title> — <YYYY-MM-DD>

| | |
|---|---|
| **Spec** | <docs/specs/<file>.md, or "n/a (internal) — <reason>"> |
| **Status** | <Draft \| Approved <date> \| In progress — <phase> next \| ✅ Completed <date>> |
| **Trigger** | <who asked for this and why, in one line> |
| **Depends on** | <plans or phases this needs first, or —> |
| **Touches** | <the paths this will change> |
| **Open questions** | <OQ-n, or "none"> |

<!--
Keep the Status row current. It is the first thing anyone reads, and a plan
whose phases are all done but whose status still says "In progress" is the
single most common form of drift in docs/. See
"When a phase's tasks are fully completed" for exactly when to update it.
-->

## Summary

<!--
Two or three sentences: which spec this serves, and the technical approach in
one breath. Someone should be able to read only this and know roughly what is
being built.

Do NOT re-explain why the owner wants it. Link the spec and move on.
-->

## What the spec asks for that isn't obvious

<!--
DELETE if the spec translated cleanly.

Where the spec's plain-language ask turns out to be technically load-bearing,
surprising, or in tension with what exists. "The owner wants the machine list
to show which one is theirs at a glance" sounds trivial and means the daemon
must report an identity the cloud does not currently store.

This section is what stops a plan quietly solving an easier problem than the
one that was asked for.
-->

## Work breakdown

<!--
Split the work into FOUNDATIONAL and PER-STORY. This split decides how tasks
get grouped, so getting it right here saves an argument later.

THE TEST — apply it honestly, per item:

    Can the owner SEE the result of this work?
      yes → it belongs to a user story
      no  → it is foundational

Foundational is schema, protocol, transport, migrations — real work that
serves every story and demos to nobody. It gets ordinary technical tasks and
BLOCKS the story work behind it.

Per-story work is anything the owner opens and uses. It gets tasks grouped by
story, each group ending in something demoable.

THE FAILURE MODE THIS PREVENTS: everything gets called foundational, no story
ever ships, and the app is a backend with no way in. If this table has stories
with no rows, that is the warning sign — not a scheduling detail.
-->

### Foundational — blocks all stories

| Work | Why no story owns it |
|---|---|
| <work item> | <what makes it invisible to the owner> |

### Per story

| Story | Work | Delivers |
|---|---|---|
| <US1> | <work item> | <what the owner can do once this lands> |

## Decisions

<!--
The load-bearing technical choices, each with its reasoning. This is the most
valuable section in the file — six months from now the code shows WHAT was
built, and only this shows why the alternatives were rejected.

Format that works well: a bold claim as the lead sentence, then the reasoning
underneath. One paragraph per decision, ### headed if there are more than
about four.

State rejected options explicitly. "We chose X" is much weaker than "we chose
X over Y, because Y would have meant Z."
-->

## Phases

<!--
DELETE THIS SECTION for a single-shot plan.

One subsection per phase. Each carries its own checklist, files, and verification, since
decomposed. Say what the phase delivers and what it depends on — not how, that
is the task documents' job.

Phases inherit the split above: a foundational phase, then one phase per story
(or one phase covering several small stories). Say which each is.

Mark phases done in place as they land (### Phase 2 — <name> ✅ DONE <YYYY-MM-DD>)
rather than deleting them.
-->

### <M-n> — <name> <(foundational | serves US1)>

<!-- What it delivers, what it depends on, and how you'll know it's done. -->

## Scope boundaries

<!--
What this plan deliberately does NOT do, and where each excluded thing went.
Naming the boundary is what stops the next agent "helpfully" building it.

Anything parked here gets a real entry: Deferred.md if it's agreed and
postponed, Ideas.md if it was merely noticed. Link the id.

The spec's own Assumptions section may already name some of these — cite it
rather than repeating.
-->

## Verification

<!--
How anyone will know this plan worked. The spec's Success Criteria (SC-nnn)
are the bar; this section says how each gets checked and by whom.

Every story's acceptance scenarios must be reachable. If part of it can't be
verified (no deployment, no second machine, the platform won't deliver the
signal), say so HERE rather than discovering it at the end — that is what
KnownGaps.md is for, and naming it early is what stops a phase quietly grading
itself on the half it could reach.
-->

| Spec criterion | How it gets checked |
|---|---|
| <SC-001> | <the concrete check> |

## Result

<!--
Filled in as the plan lands — what shipped, what was found while building that
the plan didn't anticipate, and what it spawned into the registers.

Say which of the spec's stories are actually usable now. That, not the task
count, is what the owner asked for.

The "what was found" part matters more than it looks. Every phase in this repo
so far has turned up at least one load-bearing thing that was invisible from
the plan's bullet list; writing those down is how the next plan gets written
less naively.
-->
