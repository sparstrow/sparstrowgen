<!--
TEMPLATE — these four are ENTRIES APPENDED to existing files, not new files.
Copy the relevant block into the right register and delete the guidance.

Picking the right register is most of the work:

  | The situation                              | Register          | Id   |
  |--------------------------------------------|-------------------|------|
  | "I'm not answering that right now"         | OpenQuestions.md  | OQ-n |
  | "Let's do that later" (agreed, parked)     | Deferred.md       | D-n  |
  | "It's built, but I couldn't prove it"      | KnownGaps.md      | G-n  |
  | "It works, but only within these limits"   | KnownGaps.md      | G-n  |
  | "Might be nice one day" (merely noticed)   | Ideas.md          | I-n  |

Deferred vs Ideas: Deferred entries have a DECISION behind them and a trigger
for unparking. Ideas were merely noticed and may never be built.

KnownGaps vs bug/: a gap is a statement about the strength of the EVIDENCE
behind working code — not a claim that anything is broken. If something is
actually behaving wrong, it is a bug.

Use the next free number; ids are never reused, even after an entry is
deleted.
-->

---

# OQ-n — OpenQuestions.md

<!--
Format is mandated by AGENTS.md §4 and is not optional: context, a plain
user-side scenario, then options each carrying pros/cons, a score, blast
radius if chosen wrong, caveats, and a recommendation.

A question with no options is not ready to be asked. Presenting one is asking
the owner to do the analysis you were supposed to do.

An open question blocks ONLY the checklist item that depends on it — never the
whole task, never the plan. Mark that item `[~] blocked → OQ-n` and build
everything else.

WHEN ANSWERED: record the answer in the plan or task that consumes it, then
DELETE the entry from OpenQuestions.md. That file only ever holds what is
still open, so its length is a real signal.
-->

## OQ-n — <the question, as a noun phrase>

**Raised:** <YYYY-MM-DD>, <during what>.
**Blocks:** <the specific checklist item(s) — never "the task">

### Context

<!-- What makes this a real question. What is true today, and what forces a choice. -->

### Scenario

<!--
A plain, concrete situation in the owner's own terms — no jargon, no internals.
"An agent spends 40 minutes refactoring on your desktop, then the machine
reboots. What survives?" This is what makes the question answerable by someone
who has not read the code.
-->

### Options

<!--
Open with one line saying which moment from the Scenario above every option
replays. That sentence is what turns a list into a comparison.
-->

**A — <name>**
- **What this is:** <what this option actually IS — what gets built or
  configured, what the owner has to do, what changes. Concrete enough to tell
  it apart from B and C.>
- **<The scenario's moment>, under A:** <the SAME person in the SAME situation
  as the Scenario section, and what happens to them under this option. Not a
  new example — the same one, replayed. This is the field the owner actually
  answers from; without it they are comparing adjectives.>
- **Pros:** <what it buys>
- **Cons:** <what it costs>
- **Score: <n>/10**
- **Blast radius if wrong:** <what breaks, and whether it's recoverable>
- **Caveats:** <what must be true for this to work>

**B — <name>**
<!--
Same seven fields, and crucially the same replayed moment — if B's scenario
features a different person doing a different thing, the two options cannot be
compared and the reader is back to guessing.

Two options minimum; three is usually the honest number.
-->

<!--
The shape to aim for: one person, performing one concrete action, replayed
once per option, with a different outcome each time.
-->

### Recommendation

<!-- Which one, and why — scoped tightly. Say what you'd narrow or exclude. -->

---

# D-n — Deferred.md

<!--
Agreed in principle, explicitly parked. Every entry needs a TRIGGER for
unparking, so nothing sits here purely because it was forgotten.

"Unpark when" is the load-bearing line. "Later" is not a trigger; "when anyone
outside the team needs email, or the app deploys publicly — whichever
comes first" is.

Write it in the same turn the owner says "park it" / "later" / "not now",
rather than relying on the conversation being re-read.
-->

## D-n — <what is being parked>

**Parked:** <YYYY-MM-DD>, by <the owner / during what> — "<their words, if they said it>"

<!--
What state this leaves things in, stated plainly. Distinguish "not built" from
"built but not switched on": an entry saying the OAuth code is
complete and verified with only configuration missing means something very
different from one saying nothing exists yet.

Name what is genuinely lost by parking, in the reader's terms. The failure
mode this section prevents: someone later assumes the parked thing half-works.
-->

- **If wrong:** <what breaks, and who notices>
- **Unpark when:** <the concrete trigger — an event, a threshold, a person>

---

# G-n — KnownGaps.md

<!--
Built, but not fully proved — or proved to be limited. This is the register an
agent reads BEFORE trusting that something works, and before claiming it does.

Add it IN THE SAME CHANGE that creates it. If a checklist item was ticked on
weaker evidence than it asked for, say so in the task's Result AND open an
entry here. A caveat that lives only in a chat message does not exist.

WHEN YOU CLEAR ONE: delete the entry and say where the proof lives. The length
of this file is a real signal — a gap lingering because closing it was
inconvenient is the exact failure this register exists to prevent.
-->

## G-n — <what is unproved, as a claim>

**Raised:** <YYYY-MM-DD> (<task or phase that left it>).

<!--
What was verified, what was NOT, and why it ended up that way. Be precise about
the boundary — a good entry says the shared shutdown handler IS
exercised through the HTTP route, and only the signal wiring is untested.

"Why it ended up that way" matters: a platform that won't deliver the signal is
a different situation from nobody having got round to it, and the reader needs
to know which.
-->

- **If wrong:** <the cost if the assumption doesn't hold — be honest in both
  directions; "cosmetic and self-correcting" is a legitimate answer>
- **Clears when:** <the concrete thing that closes it — an action someone can
  take, not "when we have time">

---

# I-n — Ideas.md

<!--
Unscoped. No commitment, no decision, possibly never built. Distinct from
Deferred: those were agreed and parked, these were merely noticed.

INVOKE THE `elaborating-ideas` SKILL. Writing a good idea entry is a
procedure, not a fill-in — establish what is true in the code today, find the
reframe, give it an arguable shape, name what it collides with, list the
decisions without answering them. This block is only the skeleton.

Length is earned by EVIDENCE, not by speculation. This template used to say
"keep them short — three paragraphs means it has become a plan"; that was
wrong, and the file's own record disproves it. Every entry that changed what
got built is long; every entry untouched since the
day it was written is one paragraph. Three paragraphs of verified current
behaviour and named open decisions is an entry doing its job. Three
paragraphs of what the feature could be is a daydream with a heading.

If an idea graduates, it becomes a spec in docs/specs/ (owner review first),
then a plan — and the entry is deleted.
-->

## I-n — <the idea, in a few words>

<!--
Sections, in this order — drop any that step 2 of the skill honestly found
nothing for, rather than padding them:

  What was noticed        the raw observation, in the owner's terms
  What is true today      verified in the code, with file.ts:line
  The reframe             what this is actually about, once you look
  A shape                 one plausible form, concrete enough to argue with
  What it touches         the other I-n / D-n / G-n / spec it overlaps, and
                          how they divide — this is what stops it being
                          built twice or closed as a duplicate
  Decisions this needs    named, ANSWERED NOWHERE. A real fork goes to
                          OpenQuestions.md as OQ-n with §4's full options
  What would make it real the event, threshold, or person. If rejected for
                          now, the condition that would revive it — that
                          sentence stops it being re-proposed unchanged

End with where it came from, italicised, when the origin explains the shape:
*Surfaced while scoring Decision 2 Option C.*
-->
