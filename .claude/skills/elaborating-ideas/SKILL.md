---
name: elaborating-ideas
description: >-
  Procedure for turning a raw noticing — owner feedback, a passing
  observation, a one-line docs/Ideas.md stub — into an I-n entry that is
  actually thinkable: what is true in the code today, the reframe that
  changes what the idea is about, a shape concrete enough to argue with,
  what it collides with, and the decisions it would need. Use when routing
  feedback to Ideas.md, when the owner says "park it" or "this is an idea",
  and before an existing thin idea is picked up for a spec. Elaborates; does
  not decide, and does not build.
---

# Elaborating an idea into docs/Ideas.md

The step before the chain everyone already knows: **this** → `writing-specs`
→ `writing-plans` → `decomposing-plans` → code. `docs/README.md`'s lifecycle
diagram has always started at `idea ──► Ideas.md`, and until now that arrow
was the only step with no written procedure behind it — so ideas got written
as one-line stubs and the thinking happened later, in chat, where it does not
survive the session.

Read `docs/Ideas.md` before writing into it, and
`docs/templates/register-entry.md`'s `I-n` block for the frame.

## What this step is for

An idea is not a small spec and it is not a bug report waiting for a
diagnosis. It is the only document in the chain whose job is to **make
something thinkable** — to take a sentence the owner said in passing and work
out what it is actually about, so that the decision to build it or drop it
can be made on evidence instead of on the sentence.

The failure this prevents has a specific shape: a raw noticing gets treated
as a yes/no question ("is this real or a misunderstanding?"), the answer is
unknown, and the item stalls — parked behind a verification nobody runs,
while the interesting part of it was never examined at all. **An idea does not
have a blocking unknown. It has an unexplored middle.**

## Refuse to decide, and refuse to build

Two hard limits, and they are the whole reason this step is separate:

1. **Elaborate, don't decide.** If elaboration reaches a genuine fork the
   owner must choose between, that is an `OQ-n` entry in
   `docs/OpenQuestions.md` with `AGENTS.md` §4's full options framework — not
   an idea that quietly picks an answer and describes only that branch. An
   idea that has answered its own open questions has become an unreviewed
   spec, which is the exact thing `docs/README.md`'s "owner reviews the spec"
   gate exists to catch.
2. **Don't build it.** No branch, no code, not even "just the small
   obvious part". `I-13` records what that costs: a first attempt at the chat
   right-click menu was built straight to code on unreviewed defaults and
   discarded unmerged, because the scope questions it answered for itself
   were the ones it got wrong.

Promotion is the owner's move, not yours. Write the entry, say what you would
recommend if asked, stop.

## Procedure

### 1. Establish what is true today — in the code, not in the description

Same standard as everywhere else in this repo (`AGENTS.md` §3.1): open the
files. An idea's entire value is that it is grounded, and an ungrounded idea
is worse than no entry at all, because it reads exactly like a grounded one.

Cite `file.ts:line` for anything load-bearing. `I-10`'s access-model table is
the worked example — six mechanisms, each with the file that implements it
and a plain sentence on what it actually covers, which is what made "the half
that is urgent is not the half that feels urgent" a defensible claim rather
than a slogan.

Look one layer past the obvious surface. The observation names where the
owner noticed something; it rarely names where the thing lives.

### 2. Find the reframe

**This is the deliverable.** If the elaboration ends where the observation
started, nothing has been elaborated — it has been transcribed, and the
one-line stub would have been cheaper.

The reframe is what becomes true once step 1 is done, and it is usually one
of these:

| Shape | What it sounds like | Example |
|---|---|---|
| The question was wrong | "the interesting question is not X but Y" | `I-10` — agents, not people, are the subject with real exposure today |
| The observation is a symptom of something structural | "this cannot work anywhere, not just here" | a rendering gap that turns out to be four layers each independently lossy |
| It is already half-built | "the mechanism exists and nothing reads it" | `I-8` — `users.bio` and `workspaces.context` are stored, displayed, and consumed by nothing |
| The owner's instinct fits the architecture better than the literal ask | "the second thing they asked for is the right shape for the first" | |
| It is blocked on something that just landed | "the thing this waited for shipped" | `I-11` — folder browsing waited on the live channel M16/M17 built |

If none of these appears after honest looking, say so plainly and keep the
entry short. A verified "this is exactly what it looks like" is a legitimate
result; an invented reframe is not.

### 3. Give it a shape concrete enough to argue with

Not a design, and not four options. One plausible form, named specifically
enough that the owner can say "no, not like that" — which is the response
that actually moves an idea forward.

`I-15` is the calibration: an aggregate `n/total online` pill sourced from the
`machineState()` query `machines.tsx` already computes, reusing its existing
dot vocabulary, click-expands to a per-machine popover, zero machines is
neutral not red. That is four sentences and it is arguable. "Improve the
header status indicator" is not.

Reuse beats invention here — naming the existing query, component, or
vocabulary a shape would sit on is most of what makes it credible.

### 4. Connect it to what already exists

Search `docs/Ideas.md`, `docs/Deferred.md`, `docs/KnownGaps.md`,
`docs/OpenQuestions.md`, and `docs/specs/README.md` for the same territory
before writing a new entry. Then say explicitly what this overlaps and how
they divide.

Two entries silently describing the same work is how a thing gets built
twice, and `I-10` shows the fix: it carries a whole paragraph on why the
access spec it spawned **is not this idea** — that spec decides what access
means, I-10 remains the surface that renders it. Without that paragraph the
next reader closes one of them as a duplicate.

If the overlap is total, extend the existing entry instead of adding one.

### 5. Name the decisions, answer none of them

List what the owner would have to settle, in their terms, with enough context
that each is answerable — and stop there. `I-13` names three (does Delete
exist at all; are chats pinnable given pins are localStorage-only; how large
a shortcuts page should be when two shortcuts exist) and answers none.

If one of them blocks *writing the entry* rather than building it, promote
that one to `OQ-n` per the refusal above and keep writing around it.

### 6. Say what would make it real work, and what would kill it

Both directions, concretely:

- **What would make it worth doing** — the event, threshold, or person.
  `I-14`'s is "enough unpair/re-pair churn that the Auth → Users list becomes
  hard to read, or an audit that wants the count exact", and it says plainly
  that neither is true today.
- **What would kill it** — if it was considered and rejected, the condition
  that would revive it. `I-4` is rejected for giving up in-place work on a
  dirty checkout, and names the case that would bring it back (per-project,
  for sandbox projects). That sentence is what stops the same proposal
  arriving again unchanged in three months.

### 7. Close out

- Append as the next free `I-n`. **Ids are never reused**, including after an
  entry is deleted.
- End with the origin, italicised — *Surfaced while …* — whenever the origin
  explains the shape.
- If the idea came from `docs/feedback/`, fill in that item's **Triage**
  pointing here and flip it to `🟢 routed` in the same change, index row
  included. An idea is a real destination, not a way of leaving feedback
  untriaged.
- If step 1 turned up something behaving *wrong* rather than merely absent,
  that half is a `docs/bug/` file, written the same turn (`AGENTS.md` §5).
  An idea and a bug can come out of one observation; they do not merge.

## On length

`docs/templates/register-entry.md` used to say "keep them short — an idea that
needs three paragraphs has probably become a plan." That advice was wrong,
and the repo's own record is the evidence: every entry that changed what got
built is long (`I-10`, `I-13`, `I-14`, `I-15`), and every entry that has sat
untouched since the day it was written is a single paragraph.

The real rule is that **length must be earned by evidence, not by
speculation.** Three paragraphs of verified current behaviour, a named
collision, and four open decisions is an entry doing its job. Three
paragraphs of what the feature could be is a daydream with a heading, and
would have been better as one line.

Short is still right when step 2 honestly finds nothing — the cost of a thin
entry is that it gets re-derived later, which is cheap. The cost of a
confident, ungrounded one is that it does not.
