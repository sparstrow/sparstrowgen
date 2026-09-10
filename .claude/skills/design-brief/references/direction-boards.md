# Direction boards

A direction board is a single self-contained HTML page showing **the same
realistic screen fragment rendered two or three different ways**, side by side,
so the owner can point instead of describe.

This is the mechanism that makes the interview produce real answers. Without it
you are asking a non-designer to specify a design in words, which produces
adjectives, which produce a generic doctrine.

## The rules that make a board work

**Hold the content constant.** Same rows, same labels, same numbers in every
direction. The moment the example content differs, the owner is comparing the
examples rather than the designs, and they will pick the one with the nicer
sample data without knowing that's what they did.

**Use the product's real domain.** Real-looking machine names, order numbers,
customer names, dates. Never `Lorem ipsum`, never `Item 1 / Item 2`. Placeholder
content hides exactly the problems worth catching now — how a long name
truncates, whether four-digit figures still align, how a status reads at a
glance.

**Show colour applied, never as swatches.** A palette as a row of chips tells
you almost nothing about how it behaves as a page. Put it into an actual list
row, a button, a badge, a header. Palettes that look great as chips routinely
collapse in use, and the reverse happens too.

**Label each direction with a name, not a number.** "Dense & technical" /
"Calm & spacious" / "Bold & product-led" gives the owner language to react with.
"Option A / B / C" gives them nothing, and their feedback will be correspondingly
vague.

**Two or three, never five.** Beyond three, comparison stops being a choice and
becomes a survey. If there are genuinely five candidates, run two rounds.

**Make the differences real.** Three directions that differ only in accent hue
waste the round. Vary the things that actually determine character: density,
depth strategy, type contrast, icon presence, colour usage.

## What to put on a board

One board per question, roughly in this order — each later question is easier
once the earlier ones are settled:

| Board | What varies | Typical directions |
|---|---|---|
| **Character** | The whole feel at once | dense/technical · calm/spacious · bold/product-led |
| **Colour** | Palette, applied | neutral+one accent · two-tone · saturated |
| **Depth** | How elevation reads | flat + 1px border · soft shadow · layered surfaces |
| **Density** | Row height, padding, type size | compact · comfortable · roomy |
| **Icons** | Presence and weight | none · functional only · icon-forward |
| **Motion** | Duration and easing | none · subtle · expressive |

Run the **character** board first. It usually settles half the later questions
implicitly, and the owner's reaction to it tells you which of the remaining
boards are even worth building.

**Build the shared fragment out of the elements the owner said they expect**
(SKILL.md Step 2). A board rendering a generic card teaches you about cards; a
board rendering *their* list row with *their* side panel open teaches you about
their app — and it doubles as the first honest look at whether the inventory
they described actually holds together. When an expected element is unusual or
easily confused with another, give it a board of its own rather than a
paragraph.

## Building one

Write a single `.html` file with no external dependencies — inline CSS, inline
data, no build step, no network. It has to open from disk and it has to survive
being reopened next week.

Structure that works well: a column per direction, a shared caption row naming
each, and the same fragment repeated down each column. Give each direction its
own CSS custom-property block scoped to its column so the fragments are
genuinely styled differently rather than differing by hand-written overrides.

Put boards somewhere temporary and obviously disposable — a scratch directory,
or `design-brief/` at the repo root if the owner wants them kept. They are
decision aids, not deliverables. Once `DESIGN.md` is written the board's job is
done; the doc is what carries forward.

**Then actually open it and look at it before showing the owner.** A board with
a broken layout wastes the owner's round and, worse, gets read as the design
being bad rather than the board being broken.

## Presenting it

Show the board, then ask a question that has a concrete answer:

- Good: "Which of these three reads closest to what you pictured?"
- Good: "Anything in the one you picked that you'd change?"
- Weak: "What do you think?" — invites adjectives, which is what we're avoiding.

When the owner picks, **ask what specifically made them pick it.** The reason is
worth more than the pick, because the reason generalises to the rest of the
document and the pick only covers this one board. "The rows are easier to scan"
becomes a density rule; "I liked it" becomes nothing.

Mixed reactions are normal and useful — "this one's spacing, that one's colour"
is a real answer. Build the combination and show it again rather than asking
them to imagine it.
