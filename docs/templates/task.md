<!--
TEMPLATE — copy to docs/tasks/<phase>/T-<phase>-<nn>-<slug>.md, then delete
every HTML comment in the copy.

THE BAR (docs/tasks/README.md): a task is ready when someone can work it
without asking the owner anything.

  - Every decision it needs is already made and written down, with reasoning
  - The exact files to create or change are named
  - Traps and failure modes are called out before they're hit
  - Verification is concrete enough to run, with unambiguous pass/fail

A task document contains ZERO open questions. If writing one surfaces a
genuine question, it goes to ../OpenQuestions.md and blocks ONLY the checklist
item that depends on it — see the Checklist section below.

Before writing: read ../../KnownGaps.md, so this task doesn't inherit an
unproved assumption as a fact.
-->

# <T-id> — <short name>

| | |
|---|---|
| **Tag** | `<[S] sequential \| [P] parallel \| [C] concurrent>` — <why, e.g. "shares main.go with T-02; one worker at a time"> |
| **Serves** | <`US-n` — <the story, in a few words> \| **foundational** — <the story phase it unblocks>> |
| **Depends on** | <task ids, or —> |
| **Blocks** | <task ids, or —> |
| **Phase spec** | [README.md](README.md) |
| **Status** | <not started \| in progress \| ✅ done <date> \| done except OQ-n> |

<!--
SERVES is the traceability line: every task points either at the user story it
delivers or at the story work it unblocks. A task that can name neither is a
task nobody asked for — check it against the plan's Work breakdown before
building it (AGENTS.md §3.6: build only what the plan lists).
-->

<!--
DELETE for a foundational task.

## The scenario this satisfies

Quote the acceptance scenario(s) from the spec that this task makes true —
Given / When / Then, verbatim. Verification below is graded against these
words, not a paraphrase, which is the whole point of writing them down once in
the owner's language.
-->


## Objective

<!--
What this task achieves, in two or three sentences. The outcome, not the
steps — the checklist carries the steps.
-->

## Decisions already made

<!--
Decisions settled INSIDE this task rather than inherited from the phase spec,
so a reader can tell what was decided where. Phase-level decisions get cited
("phase decision 6"), not restated.

Include code snippets when the exact shape matters — a signature, a config
constant, a schema fragment. A snippet removes a whole class of "I assumed you
meant" from the implementation.

DELETE this section if everything came from the phase spec.
-->

## Checklist

<!--
The actual units of work. Each item is something that either happened or
didn't. Tick them as they land.

  - [ ] not started      - [x] done
  - [~] blocked → OQ-n   ← blocked on an open question

A blocked item does NOT block the task. Build everything else, tick it, and
report the task as "done except OQ-n" — a real, closeable state. One missing
piece must not stop the plate being served.

Items commonly forgotten, include them when they apply:
  - typecheck and tests green for the packages touched
  - ALL FOUR STATES on any surface this task ships — populated, empty,
    loading, error. The empty state is the one that gets skipped, and it is
    the first thing the owner sees
  - a runbook row, if this task ships something only the owner can finish
-->

- [ ] <unit of work>
- [ ] <package> typecheck and tests green

## Traps

<!--
What will go wrong if someone works this task by reflex. Bold the claim,
explain underneath.

Worth writing: silent failures, things that LOOK done, shared state a
neighbouring task also touches, a name that means something different
elsewhere in the codebase.

DELETE if there genuinely aren't any — but that is rarer than it feels.
-->

## Verification

<!--
Concrete enough to run, with unambiguous pass/fail. Name the command or the
exact interaction, not "make sure it works".

If an item can only be proved by the phase's verification task, say so and
point at it — that is honest sequencing, not a gap.

If an item CANNOT be proved at all right now (no deployment, no second
machine, the harness can't render), tick nothing, say what was actually run,
and open a KnownGaps.md entry IN THIS SAME CHANGE. Shipping without proof is
allowed; shipping without saying so is not.
-->

- [ ] <assertion, with how to check it>

## On completion

- [ ] `pnpm typecheck` and `pnpm test` green
- [ ] Update this file's **Status** row — one of `queued` · `in progress` ·
      `done` · `done except <id>` · `blocked → OQ-n`, followed by the date.
      **This row is the authoritative record of this task's state.**
- [ ] Open the PR into `main` (`AGENTS.md` §2)
<!--
Add when they apply:
  - [ ] Update the phase README's task table
  - [ ] KnownGaps.md entry for anything ticked on weaker evidence than asked
-->

> **Do not edit [`../MasterTaskQueue.md`](../MasterTaskQueue.md) from a task
> branch.** Its Status column mirrors the row above; the task file's own Status
> row is the authoritative record. Sibling tasks in a phase are adjacent lines
> in one table, so a task branch ticking the queue makes a merge conflict out
> of every parallel hand-out. The queue is flipped once per phase, by whoever
> closes it.

## Result

<!--
Filled in when the task lands: what shipped, what was actually run to verify
it, and anything found along the way that the task didn't anticipate.

Name what you ran. "Verified" is not a result; "981 tests green, routes render
confirmed by clicking through from /teams, section D unreached — no deployment,
recorded as G-3" is.
-->
