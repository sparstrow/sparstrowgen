---
name: decomposing-plans
description: >-
  Step-by-step procedure for turning an approved docs/plans/*.md into the
  docs/tasks/<phase>/ folder it becomes: refuses outright if any task branch is open, then reads the shipped code first, writes the phase README, sizes
  and sequences tasks, assigns [S]/[P]/[C] concurrency tags from real file
  overlap, and regenerates MasterTaskQueue.md. Use whenever decomposing a
  plan or phase into tasks, adding tasks to an existing phase, or
  re-sequencing the queue.
---

# Decomposing a plan into docs/tasks/<phase>/

The third step of the chain, and the one that had no written procedure until
now: `writing-specs` → `writing-plans` → **this** → code.

Read `docs/tasks/README.md` and `docs/templates/README.md` before starting.
Three templates produce this folder — `phase-spec.md` for the `README.md`,
`task.md` for each task, `verification-task.md` for the `[S]` task every phase
ends with. Copy them; don't invent a shape.

## Gate: refuse if any task branch is open

This is a hard stop, not a judgment call — run it before reading the plan,
before opening a template, before writing a single file.

```bash
gh pr list --state open
git worktree list
```

`git worktree list` always shows the main checkout plus this session's own
worktree — that baseline is not a signal. What counts: **any open PR**, and
any *other* worktree whose branch is `feature/*`, `fix/*`, or `task/*` per
`AGENTS.md` §2's naming conventions (an unrelated worktree on an unrelated
branch is someone else's business, not this gate's).

**If either check turns up a live task branch, do not proceed.** Do not write
or edit anything under `docs/tasks/`, do not touch `MasterTaskQueue.md`. Stop
and report back:

- which branches/PRs are open and, if visible from the PR title or task id,
  which phase each belongs to
- that decomposition is blocked until the repo drains to zero open task
  branches
- do **not** decompose "just the parts that don't overlap" as a workaround —
  the reason below is correctness, not just merge conflicts, and it applies
  to the whole plan regardless of which files the open branches touch

Resume only once the check comes back empty. Don't ask the human whether to
proceed anyway — asking just invites a plausible-sounding reason to skip a
rule that exists because skipping it produces real defects.

Two independent reasons this is hard, not soft:

1. **Correctness.** Tasks written against a plan's outline instead of the
   shipped code are fiction in a confident tone. A phase's decomposition
   depends on what the phase before it actually turned out to look like, and
   load-bearing decisions routinely are not visible from a plan's bullet list
   at all. **An open task branch is proof the code is still moving** — not a
   maybe, the exact condition this guards against.
2. **Merge conflicts.** Decomposition regenerates `MasterTaskQueue.md` rather
   than appending to it — a whole-file rewrite that collides with every open
   branch at once.

The sequencing usually supplies the quiet moment for free: a phase is
decomposed only after the phase it depends on has landed, which is naturally
when its branches have already drained.

## Read the shipped code before writing any task

Not the plan's description of the code — the code. For each area the plan
touches, open the real files and confirm the plan's assumptions still hold.
A plan approved two weeks ago describes a repo that has moved.

Write what you found into the phase README's **The shape of what was found**
section. When it contradicts the plan, the code wins and the contradiction is
recorded there as a correction rather than folded in silently.

## One phase = one folder

`docs/tasks/<PHASE-ID>/`, tasks named `T-<PHASE-ID>-<nn>-<slug>.md`. The phase
README holds what all its tasks share, so a decision is written once and
referenced — not copy-pasted into eight files and then updated in six.

Declare the phase's **Kind** in its header: foundational (blocks stories,
demos to nobody) or serves `US-n` (ends in something the owner can open).
Every task carries a **Serves** row. A task that can name neither a story nor
the story phase it unblocks is a task nobody asked for.

## Sizing a task

One coherent unit of work, one branch, one PR. Each task must be executable
without asking the owner anything — every decision it needs is already made in
the plan or recorded in the task itself. **A task document contains zero open
questions.** If decomposing surfaces one, it goes to `docs/OpenQuestions.md`
with `AGENTS.md` §4's full options framework and blocks only the checklist
item that depends on it; the rest of the task still gets built.

Every phase ends with an `[S]` verification task built from
`verification-task.md`, graded on the spec's acceptance scenarios rather than
a list of components built.

## Concurrency tags come from file overlap, not intuition

Tag every task `[S]`, `[P]`, or `[C]` — and justify it in the task's Tag row
by **naming the shared file**:

| Tag | Use when |
|---|---|
| `[S]` | Something downstream cannot start until this lands. Gate a phase this way when siblings are written against types or names this task authors. |
| `[P]` | Genuinely no shared files with its siblings. Hand to different agents with zero coordination. |
| `[C]` | Interleavable in any order, but touches a file a sibling also touches. One agent at a time on those. |

Grep for the overlap rather than assuming it. `[P]` claimed wrongly is what
turns a parallel hand-out into a merge conflict. A large shared module that
several tasks all edit is the usual worst case — those tasks are `[C]`, never
`[P]`.

**Gate a phase on a worked example when siblings will copy a pattern.** One
`[S]` task that establishes the pattern first catches defects that would
otherwise be replicated into every file behind it.

## Regenerate the queue, don't append to it

Insert the new tasks into `docs/tasks/MasterTaskQueue.md`, re-evaluate every
unfinished task's dependencies against the new set, and reorder. A task
already `in progress` keeps its slot; anything still `queued` may be
resequenced.

State the cross-phase collisions explicitly in the phase's prose — which tasks
in other phases touch the same files, and what the coordination point is. That
paragraph is what makes a parallel hand-out safe, and it is the part a reader
cannot reconstruct from the rows.

**A dependency may point at a phase that's been archived.** Fully-done phases
move out of `MasterTaskQueue.md` to `CompletedMasterQueue.md`, leaving a
one-line stub — check both files before assuming a phase's dependency chain
isn't in the active queue.

## Don't decompose what isn't real yet

A phase whose shape depends on a phase still being built stays **not
decomposed**, and the queue says so in a row of its own. Writing its tasks
early is the failure this whole file exists to prevent. Name the dependency
and move on.
