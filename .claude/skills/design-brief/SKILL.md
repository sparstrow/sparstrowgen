---
name: design-brief
description: >-
  Interviews the owner — showing them real rendered options, not just asking
  questions — establishes which surfaces and elements the app is expected to
  contain, and writes DESIGN.md, the project-specific design doctrine every
  other frontend agent then obeys. This is the FIRST skill in the design chain:
  run it before any design system, prototype, component, or UI code exists.
  Use it whenever an app needs its look decided, whenever the current design
  feels generic, plain, blunt, or "not what I pictured", whenever a DESIGN.md
  exists that nobody actually chose (a template, a starter, another tool's
  output), and whenever the owner wants to adapt, improve, or completely
  redesign an existing app's appearance. Trigger on "set up the design",
  "what should this look like", "the UI feels off", "redesign the app",
  "create/rewrite DESIGN.md", "pick a component library", "what components
  should this app have", or any complaint about the app's visual character. Do NOT write production UI code here.
license: MIT
metadata:
  produces: DESIGN.md
  next-skill: design-system
---

# Design brief

This skill produces **`DESIGN.md`** — the document every frontend agent reads
before it designs or builds anything. Getting it right is the highest-leverage
work in the entire frontend, and getting it wrong is close to invisible.

## Why this skill exists

Agents are compliant. Hand one a design doctrine and it will execute that
doctrine faithfully, forever, across every screen — including a doctrine that
arrived from a template, a starter kit, or another tool's default output, that
nobody ever actually chose.

The failure looks like this: every screen passes review, every rule is obeyed,
typecheck is green, and the owner opens the app and finds it flat and lifeless.
Nothing is broken. The doctrine was simply never theirs. By the time anyone
notices, dozens of screens have been built correctly against the wrong
instruction, and the fix is not a bug fix — it is a re-do.

So the doc has to be **chosen**, out loud, by the person whose product it is.
That is what this skill is for.

## Step 0 — Establish the situation

Ask this first, because it changes everything downstream:

| Situation | What you're doing |
|---|---|
| **New app** | Nothing exists. Full discovery — the owner is deciding from scratch |
| **Codify** | The app looks roughly right already. Capture it so it stops drifting |
| **Improve** | The look is in the right neighbourhood but weak. Keep the spine, fix what's thin |
| **Redesign** | The look is wrong. Start over, and say plainly what is being abandoned |

For **codify** and **improve**, read the real app first — its stylesheet, its
components, a few screens — and come to the interview with observations rather
than a blank page. "Your surfaces are already three lightness steps; your
status colors are inconsistent across four screens" is a far better opening
than "what colors do you like?"

For **redesign**, be explicit about the cost before starting: every existing
screen becomes off-doctrine the moment the new doc lands. That is sometimes
exactly right, but the owner should choose it knowing.

## Step 1 — Get references before asking anything abstract

**Ask what apps or sites they want this to feel like, early.** One good
reference carries more information than twenty abstract questions, because it
encodes a hundred decisions at once that nobody has vocabulary for.

Then **actually go look at the references.** Fetch them, screenshot them, read
their type and spacing. Do not work from memory of a brand — memory produces a
generic impression of the brand, which is precisely the failure this skill
exists to prevent. Report back what you actually observed, and check it against
what the owner thought they were pointing at; those differ more often than you'd
expect.

Useful things to pull from a reference, in rough order of value: information
density, type scale and weight contrast, how much colour is used and where,
whether depth comes from shadow or border, corner radius character, icon
presence and weight, and how much motion there is.

## Step 2 — Ask what the app is expected to contain

`DESIGN.md` is a frontend instruction, not a mood board. Every agent that
builds a screen reads it first — so if it says how things should *look* but
never says what the app is *made of*, each agent invents its own inventory. The
result is an app that is on-doctrine screen by screen and incoherent as a
whole: one page grows a side panel, the next a modal, the third a full page,
and all three pass review.

**So ask it directly, in the owner's words:**

> What design elements, surfaces, or components do you already picture this app
> having?

Real answers sound like: *"the user should be able to switch the surface
character"*, *"they pick their own brand colour"*, *"profiles open in a side
panel inside the profiles tab"*. Those are three separate, load-bearing
instructions that no colour or type question would ever have surfaced.

**If the owner has no answer, do not move on and do not fill it in yourself.**
That vacuum is exactly what this skill exists to prevent. Give them enough
context to have an opinion: walk the grouped menu in
[references/element-inventory.md](references/element-inventory.md), which
carries, for each element, what it is, when it earns its place, and what it
costs. Show two or three groups — not the whole list — and let them sort into
**expect / not building / not yet**.

The forcing questions in that file are for owners who go blank on the abstract
version. "When you click a row, what should happen — a page, a panel, or a
modal?" gets an instant answer from someone who could not have told you they
wanted an inspector panel.

**Separate the mechanisms that get conflated.** When the owner names two things
that sound alike, make each answer a different question out loud before writing
either down. This is not hypothetical: this repo's own doctrine records a tab
strip (*which entity is open?*) and a side sub-nav (*which section of this
entity?*) as "conflated in the original ask" — caught mid-build, not at
interview time, because there was no step that asked for the inventory.

Carry the result forward as three lists — expected, explicitly not, undecided —
into the doc's expected-surfaces section. **"Not building this" is as
load-bearing as "building this"**: an agent that knows the command palette was
rejected won't propose one every third screen.

**Then ask once more at the close** (see the interview guide's closing sweep).
Owners recognise a missing expectation when they read the draft far more
reliably than they recall it cold at the start.

## Step 3 — Show, don't ask

**This is the technique that makes the interview work.** Most owners cannot
answer "what radius scale do you want" — and shouldn't have to. Almost everyone
can answer "which of these three?" instantly and correctly.

So for every decision that can be seen, **build it and show it**. See
[references/direction-boards.md](references/direction-boards.md) for how to
construct these.

Two rules that matter more than they look:

- **Show the palette applied, never as bare swatches.** A colour set looks
  completely different as a row of chips than it does as an actual list row,
  a button, and a status badge. Swatches flatter palettes that fail in use.
- **Render the same realistic screen fragment in each direction.** Holding the
  content constant is what makes the comparison about the design rather than
  about the example.

Reach for words only where a picture genuinely cannot carry it — priorities,
audience, what the product must never feel like.

## Step 4 — The component library is a question, not an assumption

Ask which component library the project should use — shadcn/ui, Hero UI,
Mantine, Park UI, Radix primitives directly, or none — and let the owner
choose. Do not inherit the answer from whatever the repo happens to contain.

For an existing app, say what is already installed and what switching would
cost. For a new app, the honest framing is that this choice determines the
component vocabulary `DESIGN.md` can reference and how much bespoke work every
screen needs, so it is worth two minutes now.

Once chosen, **check what that library actually ships** before writing the doc,
using the library's own registry or MCP tools rather than assumption. A doctrine
that references a component the library doesn't have sends every future agent
into building it by hand.

## Step 5 — Converge, and don't stop early

Loop: propose → show → get reaction → revise → show again. **Keep going until
the owner says it's right**, not until you have enough material to write
something plausible.

Signs you are not done: the owner's feedback is still adjectives ("cleaner",
"more modern"); they have not seen a single rendered option; you are about to
write a section by inference rather than from something they said or picked.

**Convert every adjective into a decision before the doc is written.** "Clean"
is not an instruction an agent can follow — but it usually decodes, with one
follow-up, into something like "more whitespace, fewer borders, one accent
colour." Adjectives that survive into `DESIGN.md` get reinterpreted freshly by
every agent that reads them, which is how a consistent design slowly stops
being one.

If a single decision genuinely cannot be resolved — the owner has no opinion
yet, or it depends on something unbuilt — **park that one decision and finish
the rest.** In a repo with an open-questions register, record it there. One
undecided radius must never hold up the whole document.

## Step 6 — Write DESIGN.md

Use [references/design-md-template.md](references/design-md-template.md).

Three sections are mandatory and are the three most often missing:

- **Expected surfaces & elements.** The Step 2 inventory, written down as
  expected / not building / not yet. Without it the doc describes how the app
  should look and never what it is made of, so every agent supplies the missing
  half itself and the app ends up coherent per screen and incoherent overall.
- **Iconography.** Which icon set, what sizes, what stroke weight, and — most
  importantly — a semantic map: which icon means which concept. An app whose
  icons are chosen ad-hoc per screen reads as unfinished no matter how good the
  type is, and "no icon rule" reliably becomes "no icons anywhere," which is
  usually not what anyone wanted.
- **Motion.** Durations, easing, what animates and what must not. Absent a
  motion section, agents ship zero motion, and the result feels dead in a way
  owners describe as "plain" without being able to point at the cause.

Beyond those, the thing that separates a useful doctrine from a decorative one:

**Every rule must be checkable.** Someone — possibly an automated conformance
pass — has to be able to answer "does this screen comply?" without judgment
calls. "Accent colour on at most 10% of a screen" is checkable. "Restrained
use of colour" is not, and will be obeyed differently by every agent that
reads it.

**Say what is deliberately undecided.** A section marked "not decided yet — ask
before inventing" prevents an agent from quietly filling the vacuum and having
its guess become the de-facto standard.

## Step 7 — Hand off

Once the owner signs off:

1. `DESIGN.md` lands at the repo root (or wherever the project's agent
   instructions point).
2. Run the **`design-system`** skill to turn the doctrine into a browsable
   system and `index.html`.
3. Prototypes (`interactive-prototype`) and real UI come after that, never
   before — they are accountable to the doctrine, and building them first means
   building them against nothing.

**Check for competing copies of the old doctrine before you finish.** Design
rules have a habit of being restated inside conformance checkers, agent
definitions, linters, machine-readable exports, and component comments. Any of
those that hardcode the previous design will keep enforcing it after
`DESIGN.md` changes, silently overriding the new doc. Grep for the retired
rules by name and re-point whatever you find at `DESIGN.md` instead of
restating it.

**Log the founding decisions.** Write the significant choices — and what was
rejected on the way — into `design-system/DECISIONS.md`, per the
`design-system` skill's decision-log reference. This is the record that lets a
future reader tell a decision from a default. Its absence is precisely what
allowed an unchosen doctrine to govern the app in the first place: every rule
looked equally authoritative, because nothing recorded which had actually been
argued for.

## Scope boundaries

- **No production UI code.** This skill produces a document. Building against
  it is `frontend-wiring`'s job, and seeing it is
  `interactive-prototype`'s.
- **Don't decide for the owner.** Recommending strongly is good; picking
  silently because they were vague is how the original problem happened. If
  they won't decide, park it visibly rather than inventing.
- **Don't write a doctrine you can't point at evidence for.** Every rule should
  trace to something the owner said, picked, or referenced — or be flagged in
  the doc as your recommendation awaiting confirmation.
- **Don't preserve a rule just because it was there.** In codify/improve mode
  the temptation is to transcribe the existing app faithfully. Ask whether each
  inherited rule was ever actually chosen; the ones that weren't are exactly
  what the owner is trying to escape.
