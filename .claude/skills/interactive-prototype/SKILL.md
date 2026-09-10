---
name: interactive-prototype
description: >-
  Builds a clickable, high-fidelity HTML prototype of a page, screen, or feature
  before any production code is written, using the project's design system
  tokens and components, and lands it in design-system/designs/ with a preview
  card and a handoff contract. Use this skill whenever the user wants to see,
  try, click through, or validate a feature before building it — including
  "prototype this", "mock up this screen", "what would this look like", "build
  me a clickable version", "let's see it before we build it", or when they hand
  over a spec and want it visualised. Also use it to explore several layout
  directions for one screen side by side. Do NOT use it to build production
  application code.
license: MIT
metadata:
  companion-skill: design-system
  produces: design-system/designs/<Category>/
---

# Interactive prototype

A prototype here is a **single self-contained `.dc.html` file you can click
through** — real layout, real interactions, real-looking data, no build step and
no backend. It exists to answer "is this right?" while changing it still costs
minutes instead of days.

It lands inside the design system rather than a scratch folder, because a
prototype built against ad-hoc colours drifts from the product immediately and
teaches you the wrong thing. Read
[`../design-system/references/file-conventions.md`](../design-system/references/file-conventions.md)
for the folder contract this writes into.

## First: is there a design system?

Look for `design-system/system.json`.

- **Present** → read it, read `design-system/README.md`, and build against those
  tokens and components. This is the normal case and the whole point.
- **Absent** → say so, and offer to run the `design-system` skill first. You
  *can* prototype without one, but everything you produce will be invented
  styling that has to be re-decided later. Prototyping first is legitimate when
  the product's look genuinely does not exist yet — in that case the prototype
  becomes the raw material for the design system, and you should say that
  explicitly rather than quietly inventing a palette.

## Two modes

The mode determines what the prototype is *accountable to*, and it changes how
much you are allowed to invent.

### `build` — from a spec

The user points at a written spec (in this repo, `docs/specs/*.md`). The
prototype's job is to render **what the spec actually says**, so the owner can
see whether the spec describes what they wanted.

- Read the spec fully, including acceptance scenarios, the four required states,
  and anything marked `[NEEDS CLARIFICATION]`.
- Every user story and acceptance scenario should be walkable in the prototype.
  A scenario you cannot click through is a scenario nobody has actually reviewed.
- **Do not resolve the spec's open questions by picking one silently.** Where the
  spec is undecided, either show both options side by side or render the state
  visibly marked as a guess. A prototype that quietly answers an open question
  gets treated as the decision.
- Anything you add that the spec does not mention goes in the handoff's
  "invented" list. This list is the single most valuable thing you produce,
  because those are the decisions nobody has approved yet.

### `explore` — from a prompt

No spec exists; the user is thinking out loud. The prototype is how they find out
what they want.

- Sketching is legitimately how a spec gets discovered, so do not demand one.
- Prefer **two or three genuinely different directions** over one polished
  answer. Different framings — dense vs spacious, wizard vs single-page,
  list-first vs detail-first — surface the real preference far faster than
  iterating on a single guess.
- The output is still real and clickable; "exploratory" describes its
  accountability, not its quality.
- Mark it `Provenance: exploratory — no spec` in the handoff, so it is never
  mistaken for approved scope.

## What it needs from the user

Only the first is required — infer the rest and state what you inferred.

| Input | Required | Notes |
|---|---|---|
| **What to prototype** | yes | A spec path (`build`) or a description (`explore`) |
| Surface name | no | Defaults from the spec title; becomes the filename and card title |
| Category | no | The `designs/<Category>/` nav group. Reuse an existing one before inventing |
| Fidelity | no | `static` (layout only) or `live` (clickable, stateful). Default `live` |
| Data | no | Reuse `design-system/lib/*-data.js` if present — see below |
| Reference | no | An existing card/design whose shape to match |
| Viewport | no | `desktop` (default), `mobile`, or both |

If the user says only "prototype the sales orders page", that is enough: mode
is `explore`, name is "Sales Orders", everything else is inferred. Say what you
inferred in one line rather than interrogating them.

## Use the shared seed data

If `design-system/lib/*-data.js` exists, import it. If it does not and this is
the first prototype, **create it** — do not inline an array of fake rows.

This matters more than it looks. Shared seed data means a reviewer can follow one
order from the list view into the detail drawer into the invoice and it is the
same order, with the same total. Per-prototype placeholder data makes
cross-screen review impossible and hides exactly the problems prototypes exist to
catch: the customer name that only truncates at 34 characters, the total that
only misaligns at four digits.

Use plausible domain data. Never `Lorem ipsum`, never `Test User 1`. Real-shaped
names, real-shaped codes, realistic quantities and dates.

## Building it

1. **Read** `system.json`, `README.md`, and the cards for the components you will
   use — especially their `.prompt.md` files, which say what each variant *means*.
   Also check `DESIGN.md` §6 (Iconography): does any value on this screen have a
   unique, externally-recognizable identity that should carry its own mark rather
   than text (DD-016)? Does any control trigger an irreversible action, which
   changes its resting colour, not just its hover/confirm state?
2. **Create the files** via the design-system CLI so the folder contract and the
   manifest stay correct:
   ```bash
   node .claude/skills/design-system/scripts/ds.mjs add \
     --root design-system --kind prototype --name "Sales Orders" --category "ERP App"
   ```
3. **Build the `.dc.html`.** One file. Link `../../styles.css` for tokens. Inline
   the interaction JS. No bundler, no npm install, no network calls — it has to
   open from disk in six months.
4. **Ship all four states.** Populated, empty, loading, error. Make them
   reachable — a toolbar toggle, `?state=empty`, whatever is quickest. The empty
   state is what the owner sees on day one and the one most likely to be skipped.
5. **Fill in the preview card** (`*.card.html`) so it appears in the index. A
   representative screenshot-like fragment is fine; it is a tile, not the thing.
6. **Write the handoff** — see
   [references/handoff-contract.md](references/handoff-contract.md).
7. **Rebuild it**:
   ```bash
   node .claude/skills/design-system/scripts/ds.mjs build --root design-system
   ```
8. **When the owner reacts, log why — not just what.** A prototype exists to
   provoke exactly this: "try it denser", "that colour is wrong", "add
   something here". Those requests arrive with their reasoning attached, and
   the reasoning is the valuable half — it usually generalises into a rule that
   saves every later screen from the same round of feedback. Record it in
   `design-system/DECISIONS.md` in the same turn it is said, per the
   `design-system` skill's decision-log reference. A revision applied to the
   prototype but never written down is lost the moment someone builds the real
   page from the doctrine instead.
9. **Run the `frontend-verify` skill against it before calling the prototype
   done.** This is not optional and not the same as opening it once yourself —
   that skill's loop (enumerate every state and interaction from this
   handoff, click through them, watch the console, fix and re-verify from the
   top) is what actually earns "done." A prototype nobody has run that loop
   against is still a guess, no matter how it looks in a screenshot. Append
   its report under a `## Verification` heading at the bottom of this
   prototype's `handoff.md`.

## What makes a prototype useful

- **Clickable beats pretty.** A plain screen where the tabs filter, the row opens
  a drawer, and the button changes state teaches more than a beautiful static
  image. Wire the two or three interactions the feature is actually *about*.
- **Real density.** Show twelve rows, not three. Most layout problems only appear
  at realistic volume, and three rows makes everything look fine.
- **The unhappy path.** A validation error, an empty result, a failed save. This
  is where designs are usually silent and where products feel broken.
- **Say what is fake.** If a button does nothing, make it visibly inert or label
  it. A dead control that looks live wastes the reviewer's trust once and their
  attention every time after.

## Scope boundaries

- **Never write production application code.** This skill stops at the
  prototype and its handoff. Building it for real is the app's own work — in
  this repo, the build step's.
- **Never invent tokens.** If the design system has no `--shadow-lg`, the
  prototype does not get one. Needing a token the system lacks is a finding for
  the handoff, not something to paper over.
- **Never fetch from a real API**, and never call a real backend. Seed data
  only. A prototype that needs the backend running is not a prototype.
- **Never let a prototype become the spec.** It illustrates a spec or explores
  toward one. The handoff is what carries decisions forward.
