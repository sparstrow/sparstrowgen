---
name: feedback-round
description: >-
  Captures a burst of feedback from the owner verbatim while he is still
  talking, then triages every item to a destination — bug, spec, design round,
  capability check, decision, memory, or Later — once he says he is finished.
  Use whenever he reviews something that exists and gives several items at
  once, starts numbering items, or says he is about to give feedback. The rule
  it exists to enforce: do NOT start working on item 1 while item 4 is still
  coming.
license: MIT
metadata:
  produces: docs/feedback/<date>-<surface>.md
  runs-after: any surface he can look at
---

# Feedback rounds

The owner looks at something that exists and gives several items in a row. This
is the highest-value input in the whole project — it is the only step where
taste is expressed against something real rather than imagined — and it is also
the easiest to waste.

Two ways it gets wasted, both of which this skill exists to prevent:

**Acting mid-round.** Item 4 routinely changes what item 1 meant, and work done
on item 1 lands on a surface that has already moved. Worse, replying with a fix
invites him to respond to the fix instead of finishing his list, and the last
three items never arrive.

**Letting it evaporate into chat.** The next session does not read this
conversation. And the part that decays fastest is not *what* he asked for — it
is *why*, which is carried in his own phrasing and disappears the moment it is
paraphrased into a task.

---

## Phase 1 — Capture, while he is still talking

Open `docs/feedback/<YYYY-MM-DD>-<surface>.md` on the first item and append to
it as each one arrives. Not at the end from memory — during.

**Verbatim.** Copy his words exactly, typos and all. His phrasing is the
reason, and the reason is what future sessions need. The greeking rule in
`design-shots` was reversed by nine of his words; a tidy paraphrase of them
would have changed nothing.

**He is often dictating.** Voice transcription garbles product names —
`agy` arrives as "AGI", `gemini` as "general". Keep the garbled original and
put your reading underneath it as **Reading:**. A wrong reading is then visible
and correctable instead of silently baked into the work. If a reading is a real
guess, say so and ask at triage — not mid-round.

**Keep his numbers.** He numbers his own items; use those numbers everywhere
afterwards so "go back to 4" resolves.

**Reply in a line or two, then stop.** Acknowledgement, not work, not
negotiation.

The one thing worth saying mid-round is a **consequence** — one sentence, when
an item collides with a locked decision, a capability, or another item. He
would rather know before item 7 than after the work. Say it and move on; do not
defend the existing design.

**Screenshots are evidence — extract the durable part.** He will paste a
screenshot to make a point. The image does not survive into the repo; what it
*showed* must. Transcribe the specific thing into the round file: the model
names in the picker, the actual label, the number that was wrong.

---

## Phase 2 — Triage, when he says he is finished

Every item gets a destination. Show the table, in his numbering:

| The item is | Where it goes |
|---|---|
| The app doing the wrong thing | `docs/Bugs.md` as `B-n`, then fix it |
| A specific change to something already designed | Just do it — no spec, no shot round (`AGENTS.md` §2) |
| A new capability, or a change big enough to need explaining | The approved spec — amend it or write one; **he approves before it is built** |
| A look with more than one reasonable answer | Back to `design-shots`. Do not guess a direction he did not pick |
| Something the backend may not be able to serve | `docs/Capabilities.md` — verify it and add a row **before** designing against it |
| A choice made with a real rejected alternative | `docs/Decisions.md` as `D-n`, in the same change |
| Agreed, but not now | `docs/Later.md` as `L-n`, with a concrete trigger |
| Built by this round but unproved | `docs/KnownGaps.md`, in the same change |
| How you should work, not what to build | Memory — `feedback` type, with the why |

**One item is often several destinations.** "Use real icons and drop gemini" is
a design change, an asset question, and the deletion of a parked `Later` entry.
Split it in triage rather than filing it under the closest single bucket.

**Nothing is dropped silently.** If an item is not going to be done, that is a
row in the table saying so, with the reason. An item you cannot place is a
`question` in `Later.md`, not an omission.

**Then start.** He said triage and work — do not turn the table into a second
approval gate. Proceed immediately on everything that is yours to decide, and
stop only on the items that genuinely need him: a spec to approve, a design
direction to pick, a question whose answers lead to different work. Report those
as "started everything except 2 and 5, which need you", never as "waiting for
confirmation".

---

## Phase 3 — Close the round

When the work lands, write each item's outcome into the same file — what was
done, what changed, what was refused and why. The file stops being a todo list
and becomes the record of why the surface looks the way it does.

Then ask one question of the round as a whole: **did anything here reveal a
durable preference?** If it did, it does not belong in this file, which nobody
will re-read. It belongs where it will be enforced:

- A rule about how a *kind* of work is done → the skill that governs it. The
  `design-shots` content rule was rewritten this way.
- A standing preference about working with him → memory.
- A taste note about a surface → that surface's round record in
  `docs/design/shots/<slug>/README.md`.

Most rounds reveal nothing durable, and forcing one out is how documents grow
that nobody reads. Look for it; usually find nothing.

---

## When this does not apply

- **One item.** "Make that button secondary" is not a round. Do it.
- **A conversation, not a list.** If he is thinking out loud or asking
  questions, answer them. Reach for this when he is enumerating.
- **A bug report mid-build.** Log it, fix it, carry on.
