# DESIGN.md template

The output of this skill. It is read by agents, not by people browsing — so it
is written as instruction, not description.

Every rule should be traceable to something the owner said, picked, or
referenced. Anything that is your recommendation rather than their decision must
say so inline, so a later reader can tell the difference between an agreed rule
and a plausible-sounding default.

```markdown
# DESIGN.md — <Product name>

> The design doctrine for this project. Every frontend agent reads this before
> designing or building anything. When this document and any other design
> guidance disagree, this wins.

| | |
|---|---|
| **Decided** | <date> |
| **Situation** | new \| codify \| improve \| redesign |
| **Component library** | <shadcn/ui \| Hero UI \| … \| none> |
| **References** | <sites/apps this draws from, with what was taken from each> |
| **Status** | agreed with owner \| draft pending sign-off |

## 1. North star

<One paragraph, in the owner's own framing where possible: what this product
feels like to use, who uses it, and what the interface must never do. This is
the paragraph every other rule serves — if a later rule doesn't serve it, the
rule is wrong.>

**Key characteristics:** <3–4 bullets naming the defining choices — density,
depth strategy, colour posture, type contrast.>

## 2. Colour

<Palette, with actual values. State the colour space used and why if it isn't
obvious.>

| Role | Value | Used for |
|---|---|---|
| Background | | |
| Surface / card | | |
| Border | | |
| Foreground | | |
| Muted foreground | | |
| Accent | | |
| Success / warning / danger | | |

**Named rule:** <a checkable statement about colour usage.>

## 3. Typography

Fonts, scale, weights, line heights. Each step named by role (Display,
Headline, Title, Body, Label) with the role's actual usage, not just its size.

**Named rule:** <e.g. a line-length cap.>

## 4. Spacing & layout

Base unit and scale. Page gutters, section rhythm, max content width.
Breakpoints that actually matter for this product, named.

## 5. Elevation & depth

How depth is conveyed — shadow, border, surface lightness, or a combination —
and the exceptions. If shadows are restricted to specific surface types, name
those surfaces exhaustively; "sparingly" is not checkable.

## 6. Iconography

**Mandatory section.** Its absence reliably produces an app with no icons at
all, which owners experience as "plain" without being able to name the cause.

- **Set:** <library and version>
- **Sizes:** <the two or three permitted sizes and where each applies>
- **Stroke / weight:** <value>
- **Colour:** <which token, and when an icon may take an accent>

**Semantic map** — which icon means which concept, so the same idea gets the
same icon on every screen:

| Concept | Icon | Notes |
|---|---|---|
| | | |

**Rule:** <when an icon is required, permitted, and forbidden. Be explicit about
whether decorative icons are allowed — this single line has more effect on the
app's character than any other in this document.>

## 7. Motion

**Mandatory section.** Absent this, agents ship no motion and the result feels
inert.

| Movement | Duration | Easing | Applies to |
|---|---|---|---|
| | | | |

**Rule:** <what must never animate; behaviour under `prefers-reduced-motion`.>

## 8. Component vocabulary

The chosen library, what is in use, and what is deliberately not. Name the
components an agent should reach for by default for common jobs (a list row, a
detail panel, a confirm, a status indicator) so every screen doesn't re-decide.

**Rule:** <e.g. never hand-roll a primitive the library ships; if none fits, say
so rather than inventing quietly.>

## 9. Expected surfaces & elements

What this app is made of, agreed with the owner — §8 says which components
exist to reach for, this says which ones this product actually has. Three
lists, because "not building this" carries as much instruction as "building
this".

| Element | Status | Where | Note |
|---|---|---|---|
| <e.g. side panel inside the profiles tab> | Expected | Profiles | <what it holds> |
| <e.g. user-picked brand accent> | Expected | App-wide | <see §2> |
| <e.g. command palette> | Not building | — | <why it was rejected> |
| <e.g. notifications centre> | Not yet | — | See §13 |

**Where two mechanisms could be confused, say what each answers in one line.**
That line is the reason the row exists — a tab strip answering *which record is
open?* and an in-record sub-nav answering *which section of this record?* read
as one idea in conversation and become two different builds.

**Rule:** <e.g. an agent that needs a surface not on this list adds it here
with the owner rather than inventing one on the screen it happened to be
building.>

## 10. The four states

Every surface ships Populated, Empty, Loading, and Error. State what each must
contain — particularly empty-state copy, which is the screen a new user sees
first and the one most often skipped.

## 11. Named rules

Every checkable rule from above, collected in one list so a conformance pass can
run down it. If a rule can't be phrased so that two people would agree whether a
screen passes, it isn't ready to be here.

## 12. Do / Don't

Concrete, specific, and drawn from this project's actual failure modes rather
than generic advice.

**Do:**
-

**Don't:**
-

## 13. Deliberately undecided

Decisions consciously not made yet. An agent hitting one of these asks rather
than inventing — an invented answer here becomes the de-facto standard by the
third screen that copies it.

-
```

## Filling it in well

**Write for an agent's literal reading.** Anything ambiguous will be resolved
differently by each agent that reads it, and the drift compounds silently across
screens. If a sentence could be followed two ways, it needs rewriting.

**Prefer a value to an adjective, always.** `12px` beats "tight". `140ms` beats
"quick". `≤10% of the viewport` beats "restrained".

**Don't pad it.** A short doctrine that is genuinely decided beats a long one
half-invented to fill the template's sections. Sections that were never
discussed with the owner belong under "deliberately undecided," not filled in
with a plausible default — a plausible default is exactly what this skill exists
to prevent.

**Keep the provenance row honest.** If the owner signed off on colour and type
but the motion section is your proposal awaiting review, say that in the status
line. A doc that claims more agreement than it has is worse than one that admits
what's still open.
