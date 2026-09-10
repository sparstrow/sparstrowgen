<!--
TEMPLATE — copy to docs/tasks/<phase>/T-<phase>-<nn>-verification.md, then
delete every HTML comment in the copy.

Every phase ends with one of these, tagged [S]. It exists because running the
thing finds what reading the code cannot — defects and design corrections that
a green test suite reported as fine.

What makes this different from the Verification section inside a normal task:
that one proves ONE task's work in isolation. This one proves the phase as a
product — the seams between tasks, what must NOT have changed, and the paths a
user actually takes.

The rule this task enforces: "done" must mean the same thing every time it is
written. An assertion ticked on weaker evidence than it asked for, with no
KnownGaps.md entry, devalues every other ticked box in the repo.
-->

# <T-id> — Verification

| | |
|---|---|
| **Tag** | `[S]` sequential — needs all of <phase> in place |
| **Depends on** | <the phase's other tasks> |
| **Blocks** | — |
| **Phase spec** | [README.md](README.md) |
| **Status** | <not started \| ✅ done <date> \| ⏸ not run — [`G-n`](../../KnownGaps.md)> |

## Objective

Prove the phase for real.

<!--
Then, up front, name what this pass CANNOT reach and why — a missing
deployment, a second machine nobody has, a browser pane that won't composite.

Naming it here rather than discovering it at the bottom is the difference
between a phase that reports honestly and one that quietly grades itself on
the half it could reach. Say which section is affected and where it gets
recorded (KnownGaps.md, following an existing entry's shape).
-->

## A — The acceptance scenarios

<!--
FOR A STORY PHASE: walk the spec's acceptance scenarios verbatim, Given /
When / Then, in the running app. These are the owner's own words and they are
what "done" means here — not the task checklists, which only prove the parts
were built.

Include the story's Independent test from the spec: implement-only-this-story
must still leave something usable. If the story cannot be demonstrated without
a LATER story also being present, the split was wrong — say so here rather
than quietly verifying them together.

FOR A FOUNDATIONAL PHASE: replace this section with the technical assertions,
and state which story phase is now unblocked.

Reach things the way a USER reaches them. Concretely: reach
every detail page by clicking a row, never by typing a URL with a made-up id —
a fabricated id fails exactly the way a broken param does, so typing one
proves nothing.
-->

- [ ] **US-n scenario 1** — Given <…>, When <…>, Then <…>
- [ ] **US-n scenario 2** — the unhappy path, same treatment
- [ ] The story's independent test passes with only this phase's work present
- [ ] The browser console has no errors on load

## A2 — The four states

<!--
DELETE for a foundational phase.

Per AGENTS.md §3.9 and the spec's Interface & experience section, all four
ship together. Verify each DELIBERATELY — the empty and error states are the
ones that never get looked at, because reaching them takes effort.

To reach empty: a fresh workspace, or delete the rows. To reach error: stop
the daemon, kill the network, point at a dead port. If a state genuinely
cannot be reached, that is a KnownGaps entry, not a tick.
-->

For every surface this phase ships:

- [ ] **Populated** — real data, correct record
- [ ] **Empty** — explains what to do next and offers the action, not a bare
      "No items"
- [ ] **Loading** — skeleton shaped like the real content, no layout jump
- [ ] **Error** — names what failed in plain words and offers the next action
- [ ] Both light and dark themes
- [ ] Keyboard navigation and visible focus work; nothing scrolls sideways

## B — What must NOT have changed

<!--
The regression surface. A phase that adds four routes can break the two that
already worked, and a routing mistake tends to be systemic.

Include things that are SUPPOSED to keep failing. Deliberate 501s, disabled
buttons, features that refuse by design — a helpful-looking fix to one of
those is a regression, and this is where you catch someone having "improved"
it.
-->

- [ ] <thing that worked before still works>
- [ ] <deliberate refusal still refuses, with its message intact>

## C — <what can be verified today>

<!--
Split reachable from unreachable, so the unreachable half is visibly skipped
rather than silently absorbed. A dead port, a stubbed response, or a manually
inserted row often makes "unreachable" reachable — try before deferring.
-->

- [ ] <assertion>

## D — <what needs something that doesn't exist yet>

**Needs <the missing thing>.** Skip and record if there is not one.

- [ ] <assertion that can't be reached yet>

## E — Regression surface

- [ ] `pnpm -r typecheck` and `pnpm -r test` green
- [ ] <the affected package(s) build>

## On completion

- [ ] Tick <n.n>–<n.n> in [`../MasterTaskQueue.md`](../MasterTaskQueue.md) and
      mark Phase <n> complete
- [ ] Update the phase `README.md` status line and its task table
- [ ] Update the plan's own **Status** row
<!--
If this is the plan's last phase and every phase reads done, the plan's status
becomes "✅ Completed <date>". It cannot, while any verification is unreached —
say WHICH gap is why rather than rounding up.
-->
- [ ] **Every unreached assertion above written into
      [`../../KnownGaps.md`](../../KnownGaps.md)** with what it would cost if
      the assumption is wrong, and the concrete thing that closes it

## Result

<!--
What was actually run, and what it found. Name the evidence — commands, what
was clicked, what was observed. A verification task whose Result says
"verified, all good" has failed at its one job.
-->
