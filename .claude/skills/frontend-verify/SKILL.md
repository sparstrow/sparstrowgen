---
name: frontend-verify
description: >-
  Runs the mandatory browser verification loop for any frontend work: open
  what was built, walk every state and interaction against its spec/handoff/
  design docs, watch the console and network tab (not just the screenshot),
  fix whatever is wrong at the root cause, and re-run the WHOLE checklist
  until one full pass finds nothing new. This is Definition of Done, not
  optional QA — invoke it immediately after finishing any UI-producing work
  (a new page, a component change, a bug fix) and especially right after the
  `interactive-prototype` or `design-system` skills produce something,
  BEFORE reporting the task complete. Use it even when the user didn't ask
  for testing — "it should work," a passing typecheck, and a code review
  that looks right are exactly the false signals this skill exists to catch.
  Trigger on "test this," "verify it works," "click through it," "does this
  actually work," or silently as the closing step of any frontend task.
license: MIT
metadata:
  companion-skill: design-system, interactive-prototype
  consumes: a running app route, a design-system card, or a *.dc.html prototype
---

# Frontend verify

A prototype that renders once and a feature that typechecks are both "looks
done." Neither is *done* — done means someone actually clicked every path and
watched what happened. This skill is that pass, made into something you run
the same way every time instead of reinventing under time pressure (which is
when it gets skipped).

It is not a separate QA phase you hand off to. It's the last step of the
frontend work itself, done by whoever just built it, because they're the one
who knows what "correct" was supposed to look like.

## When to run it

- The last step of any change that alters what a browser renders: a new page
  or route, a component change, a bug fix, a prototype.
- Immediately after **`interactive-prototype`** produces a `.dc.html` — verify
  against its own `handoff.md` States and Interactions tables.
- Immediately after **`design-system`** adds or edits a card — verify it
  actually renders as described, in both light and dark, including
  focus-visible and other states the card claims to show.
- After any real, production frontend change — verify against the task's
  spec/acceptance criteria if one exists, otherwise against the ordinary UX
  baseline in "No source of truth exists," below.

If nothing changed that a browser can render — a types-only refactor, a
backend change with no UI surface — this skill has nothing to do. Don't force
it.

## Inputs

| Input | Required | Notes |
|---|---|---|
| What to open | yes | A URL (dev server route) or a file (`*.dc.html`, `index.html`) |
| Source of truth | no | A `handoff.md`'s States/Interactions tables, a spec's acceptance scenarios, or the project's design docs — use whichever exist |
| Definition of "every state" | no | Enumerate it explicitly before opening the browser — see step 1 |

## The loop

1. **Enumerate the checklist before touching the browser.** Every state
   (populated, empty, loading, error — whatever the surface has), every
   interactive control, every acceptance scenario, every explicit design rule
   the project states. Write it down. This list becomes the shape of the
   report at the end — you don't get to declare done by improvising as you
   click and hoping you covered enough.

2. **Open it and take a first look** — screenshot, read the page structure.

3. **Walk the checklist, one item at a time.** For each: exercise it for
   real, then check the outcome three ways, not one:
   - **Visually** — does it look right (screenshot / read_page)?
   - **Silently** — is the console clean? A green screenshot next to a red
     console is not a passing item; console errors are often the only trace
     of a bug that happens to look fine.
   - **Structurally** — for anything that fires a network request, did the
     right one fire, with the right payload?

4. **Compare against the design system, when one exists.** Token usage (no
   ad-hoc hex colors slipping in), the component vocabulary (nothing
   reimplementing a primitive that already exists), and any explicit rules
   the project's own design doc states — read it if the repo's CLAUDE.md /
   AGENTS.md points at one; don't assume you already remember its rules
   correctly.

5. **When something is wrong, find the actual cause before fixing it.**
   "It doesn't work" is a symptom, not a diagnosis — the fix that matches the
   symptom is often the wrong fix. Read the relevant source, isolate the
   failing piece, and compare it against whatever the real implementation
   (or the spec) says should happen. See
   [references/root-cause-example.md](references/root-cause-example.md) for
   a worked example of this, including how to tell a genuine bug apart from a
   limitation of the tool you're testing with.

6. **Fix it, then re-run the WHOLE checklist — not just the item that
   broke.** A fix to one interaction routinely shifts or masks the one next
   to it (shared state, a re-render, an event listener that now fires
   twice). Spot-checking only the thing you just touched is exactly how a
   fix ships a second, quieter bug alongside it.

7. **The loop ends when one full pass finds zero new issues and zero console
   errors.** Converging in one pass is fine. Converging in five is fine.
   Declaring done after a partial pass, or after fixing the one thing you
   happened to notice, is not — that's the failure mode this skill replaces.

8. **Say what you couldn't verify.** If the test harness itself has a real
   limitation — a headless browser pane without true OS focus, a mocked
   network call, a state that needs a backend you don't have running — that
   caveat belongs in the report, explicitly, not silently absorbed into
   either "it works" or "it's broken."

## No source of truth exists

Plenty of real frontend work has no handoff and no spec — a small bug fix, a
tweak to an existing page. Verify against the ordinary baseline instead:

- Every visible control does something, and does the thing its label implies.
- No console errors or warnings introduced by the change.
- The change doesn't visibly break anything adjacent to it (check the states
  around the one you touched, not just the one you touched).
- If the project has design tokens or a documented style ruleset, the change
  follows it.

This isn't a lower bar — it's the same loop with a shorter checklist, because
there's less to enumerate up front.

## The report

State what you tested, not only what you found. An empty issues list proves
nothing if the reader can't tell what was actually exercised versus skipped.

```markdown
## Verification — <date>
**Tested:** <URL or file>, against <handoff.md | spec | design docs | ordinary UX baseline>

### Checklist
- [x] <state or interaction 1>
- [x] <state or interaction 2>
...

### Found & fixed
- <issue> — root cause: <what was actually wrong> — fix: <what changed>

### Found & not fixed
- <issue> — why: <blocked on a decision, genuinely out of scope, needs a real backend>

### Environment caveats
- <e.g. "the automated browser pane never gets real OS focus, so blur-only
  triggers don't fire here — confirmed the real interaction still works via
  its keyboard path, see below.">
```

For a prototype, append this under a `## Verification` heading at the bottom
of its `handoff.md` — it's part of the contract, and the next reader should
see it without hunting through chat history. For real app work with no
persistent artifact, put it in the response to the user; if the task already
has a doc (a bug file, a task checklist item), record it there instead so it
isn't lost the next time this conversation compacts.

## Scope boundaries

- **This skill doesn't decide what "correct" is** — it verifies against
  whatever source of truth exists, or the ordinary UX baseline when none
  does. It doesn't invent new requirements or expand what the surface is
  supposed to do.
- **Fixing stops at the bug in front of you.** A verification pass that
  surfaces a genuinely separate, out-of-scope problem reports it — flag it to
  the user, or use the project's own bug-tracking convention if it has one —
  rather than silently pulling it into the current change.
- **Don't fake the loop.** A single click with no console check, or a
  checklist item ticked without actually exercising it, defeats the entire
  reason this exists. The value here is entirely in the rigor; a shortcut
  version is worse than skipping it, because it produces false confidence
  instead of an honest "I didn't check that."
