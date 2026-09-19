---
name: design-shots
description: >-
  Generates 2-4 rendered images of genuinely different design directions for a
  screen or feature — using codex's built-in image generation — and shows them
  side by side so the owner can pick a direction in a minute instead of waiting
  for prototypes to be coded. Runs BEFORE interactive-prototype: images choose
  the direction, the prototype builds the chosen one. Use whenever a feature
  needs a look decided — "show me some options", "what could this look like",
  "give me a few directions", "I want to see it first" — or at the Design step
  of design-driven-feature. Do NOT use it to produce anything that gets built
  from directly; images are not specifications.
license: MIT
metadata:
  runs-before: interactive-prototype
  produces: docs/design/shots/<slug>/
  requires: codex CLI with image generation
---

# Design shotgun

Fire several visual directions at once, look at them, pick one. Then — and only
then — spend the time coding a prototype.

The problem this solves is ordering. A clickable prototype is the right way to
decide a design, but it costs real time to build, which quietly caps how many
directions ever get explored: two, maybe, and usually the second is a variation
on the first. An image costs about two minutes. Four images means four *actually
different* ideas got considered before anyone wrote a line of code, and the one
that gets coded is one the owner already wants.

Adapted from [garrytan/gstack's `design-shotgun`](https://github.com/garrytan/gstack/tree/main/design-shotgun).
What was taken: the anti-convergence rule, confirming concepts before spending
credits, and a side-by-side board rather than a scroll of images. What was left:
its external artifact directory, its JSON handshake files, polling loops, and a
taste profile with weekly-decaying confidence scores. This repo has one reason to
exist and it is shipping — see `AGENTS.md` rule zero.

**Named `design-shots`, not `design-shotgun`, deliberately.** gstack's skill is installed on the
owner's machine at `~/.claude/skills/design-shotgun`, and a project skill sharing that name
resolves to theirs — which would silently run their `~/.gstack` machinery instead of this. Do not
rename this back.

## What an image can and cannot decide

**Be honest about this with the owner, every time, in a line.** If they think
approving an image approved the design, the prototype step will feel like
re-litigating a settled question, and the whole ordering falls apart.

| An image genuinely shows | An image cannot show |
|---|---|
| Layout and composition | Any interaction |
| Information hierarchy — what dominates | Real component behaviour |
| Density: airy or packed | Real data: long names, empty fields, 200 rows |
| Whether the information shown is the *right* information | Whether it stays right at real volume |
| Mood, weight, temperature | The four states |
| Whether an idea is worth coding | Whether it actually works |

So: **images pick a direction, the prototype makes the decision.** Say that when
you present them.

## Feasibility first — the same gate, and it matters more here

**Read [`docs/Capabilities.md`](../../../docs/Capabilities.md) before writing a
single prompt.**

Cheap waste is still waste, and rendered waste is worse than sketched waste
because it is *seductive*: a polished image of a screen the backend cannot serve
is more likely to get approved than an honest sketch of one it can. This is
precisely the failure the whole workflow exists to prevent, and generating images
makes it faster to commit.

If a direction needs something not listed as deliverable: check it, design around
it, or cut it. Never "probably fine".

## The loop

```
Concepts    2-4 directions, one line each. Confirm before spending.
Prompts     one prompt per direction, same subject, comparable.
Generate    codex, in parallel. ~2 min and ~88k input tokens each.
Show        all of them in one turn, side by side.
Record      what won, what lost, and why. The why is the valuable half.
Hand off    interactive-prototype builds the winner.
```

### 1. Concepts — before spending anything

Write 2-4 directions as **one line each**, and get a yes before generating.

**Anti-convergence: siblings fail.** Three variations on one idea with different
spacing is one option rendered three times, and it teaches the owner nothing
except that you cannot think of a second idea. Directions must differ on
something structural:

- **Layout model** — sidebar vs top-nav vs command-palette-first vs single column
- **Information hierarchy** — what is biggest, what is hidden until asked for
- **Interaction metaphor** — a document, a terminal, an inbox, a canvas, a feed
- **Density** — a dashboard that shows everything vs one that shows one thing

If you cannot name what is structurally different about direction three, you have
two directions. Say so and generate two — a real two beats a padded three.

### 2. Prompts — the two things that decide whether this round is useful

**Write the real content into the prompt.** Real product names, real labels, real
numbers, real message text. `claude 62% resets 4:12`, not "a status indicator".
`gemini — no account signed in`, not "a disabled state". A message that actually
says something, not a bubble.

This was learned by getting it wrong. The first version of this skill said to
greek everything — ask for neutral placeholder bars, on the theory that garbled
AI pseudo-text would hijack every reaction. It does avoid that, and the images
were still useless: **the owner cannot judge a screen made of grey rectangles.**
His words, on the first round that shipped: *"I dont need empty image with boxes
and charts, I want more real and acutal image."*

A wireframe of bars answers "where do the blocks go", which is the least
interesting question and the one you can already answer in your head. A realistic
screen answers whether the thing is worth building — whether four provider chips
across the top is reassuring or oppressive, whether a cost figure per message
reads as useful or nagging. Those only appear with real content in place.

Imperfect text is an acceptable price. A slightly wrong word in a filler sentence
costs a glance; an abstract mockup costs the whole round. Filler prose is fine —
say so in the prompt — as long as it reads as real sentences rather than bars.

**Name the product-specific parts explicitly, or you get stock.** Asked for "a
chat app UI", the generator returns Telegram — verified, that is exactly what
came back. Every generic prompt regresses to the most common example of its
category, and the most common example is never the thing being designed. Spell
out what is unusual about *this* screen: the provider being switched, the limit
running out, the two agents in one thread. If nothing in the prompt is specific
to the product, the image will not be either, and the round is wasted.

A prompt that works is long — 200–300 words, region by region. Short prompts get
stock designs; this is where the specificity actually goes in:

```
A realistic high-fidelity screenshot of <app name>, a <what it is>.

Layout: <region 1 — its position, and the REAL text in it: names,
labels, values, timestamps>.
<region 2 — same treatment>.
<region 3 — same treatment, including the one state that makes this
direction different from the others>.

Style: realistic polished product screenshot of a real shipping app,
<light|dark> theme, crisp legible <sans-serif|monospace> UI text, the
quality of a developer tool marketing site screenshot. Render ACTUAL
READABLE TEXT for every label, name, number and status. Body text may
be generic filler but must read as real sentences. Absolutely no grey
placeholder bars, no wireframe rectangles standing in for text.
No logos, no watermark, no browser chrome. <viewport>
```

Walk the screen region by region and give each one its actual content. If you
cannot say what a region says, you do not yet know what the region is for — and
that is worth discovering here rather than in the prototype.

Keep style, theme, and viewport **identical across directions**. The only thing
varying is the idea. If one is dark and one is light, the owner is choosing a
theme, not a direction.

**Pull the real content from the spec and `Capabilities.md`, not from
imagination.** Provider names, the fields a provider actually emits, the states
that actually exist. An image showing a cost figure for a provider that reports
none is the undeliverable-design failure in miniature, and it is easier to commit
here than anywhere else because nobody reads a mockup sceptically.

### 3. Generate

codex has image generation built in. Run the directions **in parallel** in the
background — three serial runs is six minutes of nothing happening.

```bash
codex exec --json --skip-git-repo-check \
  "Use your imagegen skill to generate ONE image. <the prompt>.
   Save it to <absolute path>/A-<slug>.png and tell me the absolute path."
```

Notes that cost time to rediscover:

- codex writes to `~/.codex/generated_images/<thread>/<exec>.png` by default.
  Naming an absolute destination path in the prompt makes it copy the file there.
- **Do not pass `--ignore-user-config` here.** That flag is mandatory for the
  daemon's production adapter (`blueprint.yaml`, G-3's reasoning), but this is a
  build-time design tool, and the flag risks stripping the skill that does the
  work. Expect unrelated MCP auth noise on stderr; ignore it.
- One image per run. Asking for several in one turn gets one image and a
  description of the others.
- **Report every failure.** If two of four generated, show two and say the other
  two failed and why. Silently presenting three when four were promised makes the
  owner think three was the plan.

Land them in `docs/design/shots/<YYYY-MM-DD>-<slug>/`, named `A-`, `B-`, `C-`
so they can be referred to by letter.

### 4. Show them

**Send every image in one turn** (`SendUserFile` takes an array — use it once,
not once per image). Above them, one line per direction saying what it optimises
for. Then stop talking and let them look.

Do not run `AGENTS.md` §4's "presenting a choice" framework here. That framework
is for decisions with no picture. Seeing them *is* the comparison, and a
four-part written analysis of images the owner is looking at is noise.

Do not ask "which do you prefer?" as a formal question. Show them; they will say.

Expect "B's layout with C's density" — that is the normal and best outcome, not
a failure to decide. Merge and, if the merge is genuinely uncertain, fire one
more image at it rather than arguing in prose.

### 5. Record why — this is the part that compounds

Write `docs/design/shots/<slug>/README.md` in the same turn the owner reacts:

```markdown
# <Feature> — <date>

| | Direction | Optimised for | Outcome |
|---|---|---|---|
| A | <one line> | <one line> | rejected — <their reason, their words> |
| B | <one line> | <one line> | **chosen** |

## What this told us about taste
<Only durable preferences, and only ones actually stated. "Prefers the primary
action inside the content area, not a toolbar" is durable. "Liked B" is not.>
```

The reasons are worth more than the winner. A reason generalises into a rule that
saves every later screen a round of feedback; a winner only settles one screen.
When a preference shows up twice, promote it into the design system (and log it in `docs/Decisions.md`) so it
stops being rediscovered — and read this folder's earlier `README.md` files before
writing prompts, so the same rejected direction is not generated again.

### 6. Hand off

The chosen direction goes to `interactive-prototype` in `build` mode, with the
winning image as its reference. The prototype is where the four states, the real
density, and the actual interactions get decided — none of which the image
settled.

## When to skip this

- **A small specific change.** "Make that button secondary" does not need four
  images. Read the room.
- **The surface already exists** and this is one more screen in an established
  pattern. Match the pattern; the direction was chosen long ago.
- **Backend-only work.** Nothing to look at.
- **codex is unavailable.** Say so plainly and go straight to
  `interactive-prototype` with two directions. Do not simulate this step by
  describing images in prose — a described image is worse than no image, because
  it takes as long to read as looking would have taken and carries none of the
  information.

## Scope boundaries

- **Never build production code from an image.** Nothing is built from a picture
  — the prototype sits in between, on purpose.
- **Never let an image become the spec.** It has no states, no data, no
  behaviour, and no acceptance criteria.
- **Never generate a direction the backend cannot serve** — see Feasibility.
- **Never present a single image.** One image is not a choice; it is an
  instruction to approve. If only one direction is worth generating, skip this
  skill and prototype it.
