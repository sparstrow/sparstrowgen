<!--
TEMPLATE — copy to docs/specs/<YYYY-MM-DD>-<slug>.md and delete these comments.

A spec is the OWNER'S document. It says what he wants and why, in his own
terms, elaborated enough that nobody has to guess. It is the input to the
design step, and the cheapest place to catch a wrong direction.

TWO HARD RULES:

1. NO TECHNOLOGY. No tables, endpoints, component names, frameworks, or file
   paths. If a sentence couldn't be read aloud to someone who has never seen
   the codebase, it belongs somewhere else.

2. NO INTERFACE DESIGN. Do not describe layouts, screens, panels, or where
   things sit. That is what the design step is for, and describing it here in
   prose pre-empts the options the owner is supposed to choose between. Say
   what someone needs to DO and what should be TRUE afterwards; let the design
   answer what it looks like.

Keep it short. A spec that takes an hour to read costs more than it saves.
Skip it entirely for bug fixes, backend-only work, and small specific changes
("make that button secondary").
-->

# Spec: <feature name>

| | |
|---|---|
| **Status** | <Draft \| Approved <date> \| Superseded by <spec>> |
| **Created** | <YYYY-MM-DD> |
| **Trigger** | <what prompted this — the owner's words where possible> |
| **Design** | <design-system/designs/<...> once locked, or "not designed yet"> |
| **Open questions** | <L-n, or "none"> |

## What's wrong today

<!--
The current experience, concretely. What the owner does now, and where it
hurts. Name the actual friction, not an abstraction of it: "I have three
desktop windows open and re-explain the same context to each" beats "context
management is fragmented".

If this is brand new and there is no "today", say what happens instead of the
thing existing.
-->

## What I want instead

<!--
A short paragraph in the owner's voice. Not a feature list — the outcome. The
feature list falls out of the stories below.
-->

## User stories

<!--
The heart of the spec. One per thing someone needs to be able to do.

Each story must be independently useful: build only that story and the owner
still has something they can open and use. A story that delivers nothing on
its own is a technical step wearing a story's clothes — leave it out.

Priorities: P1 = the feature is pointless without it. P2 = clearly wanted,
can follow. P3 = nice, may never happen.
-->

### US1 — <short title> (P1)

**As** <who> **I want** <what> **so that** <why>.

**Acceptance**

<!--
Given/When/Then. Observable behaviour only — no mention of how it works.
Cover the unhappy paths too; those are where specs go thin, and they are what
the design's empty and error states get built from. These scenarios ARE the
verification target later — there is no separate success-criteria list.
-->

- **Given** <situation> **when** <action> **then** <what is true afterwards>.
- **Given** <the thing has gone wrong> **when** <action> **then** <what the
  person sees and can do next>.

### US2 — <short title> (P2)

<!-- Same shape. Two or three stories is normal; ten means this is two specs. -->

## Edge cases

<!--
What should happen when reality is untidy — nothing there yet, far too much,
something offline, something half-finished, two things at once. Answer in
outcomes, not mechanisms.

This section does real work: it is where the design's empty, loading, and
error states come from, and it is the part the owner is uniquely able to
answer.
-->

## Out of scope

<!--
What this deliberately does NOT do, and why. As load-bearing as the stories:
it stops the design proposing it and stops a later reader assuming it was
forgotten.
-->

<!--
Anything you are ASSUMING the backend can deliver goes in docs/Capabilities.md
and gets verified there — not written down here as a hope.

Nothing designs against a Draft. Status goes to Approved with the date once he
has actually read it and said so; "no objections" is not approval.
-->
