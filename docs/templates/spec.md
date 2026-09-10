<!--
TEMPLATE — copy to docs/specs/<YYYY-MM-DD>-<slug>.md, then delete every HTML
comment in the copy.

A spec is the FIRST document written, before any plan. It answers "what do I
want to be able to do, and what should it feel like" — never "how is it built".

  spec  → what the owner wants, in the owner's terms      ← YOU ARE HERE
  plan  → how it gets built, technically
  tasks → the executable steps
  code

WRITE NOTHING TECHNICAL HERE. No table names, no endpoints, no component
names, no framework. If a sentence could not be read aloud to someone who has
never seen the codebase, it belongs in the plan instead.

THE TEST FOR A GOOD USER STORY: implement only that one story, and the owner
still has something they can open and use. A story that delivers nothing on
its own is a technical step wearing a story's clothes — put it in the plan's
foundational work.

UNCERTAINTY IS ALLOWED HERE, exactly as in a plan. Mark it inline with
[NEEDS CLARIFICATION: <what is unknown>]. If the unknown genuinely blocks
building, promote it to an OQ-n entry in ../OpenQuestions.md with the
AGENTS.md §4 options framework — the inline marker is for "we should pin this
down", the register is for "someone must decide before this can be built".

Owner review happens after this document and before planning starts. It is the
cheapest point to catch a wrong direction — a wrong spec propagates silently
into the plan, the tasks, and everything downstream.
-->

# Spec: <feature name>

| | |
|---|---|
| **Status** | <Draft \| Owner-reviewed <date> \| Superseded by <spec>> |
| **Created** | <YYYY-MM-DD> |
| **Trigger** | <what prompted this — the owner's words where possible> |
| **Plan** | <docs/plans/<file>.md once written, or "not planned yet"> |
| **Open questions** | <OQ-n, or "none"> |

## The experience today

<!--
What using the app for this is like RIGHT NOW, including "nothing exists yet".

Be specific about what is wrong or missing from the owner's point of view, not
the system's. "The machines page lists rows but I can't tell which one is mine
without reading the hostname" is useful. "Runtime identity is not surfaced" is
the same fact written uselessly.

DELETE only if this is genuinely new ground with no current experience at all.
-->

## What I expect instead

<!--
The owner's own framing of what "good" looks like here, before it gets broken
into stories. A short paragraph. This is the sentence everything below is
graded against, and the one to re-read when a task starts drifting.
-->

---

## User stories

<!--
Prioritized as journeys, P1 first. P1 is what makes this worth doing at all —
if only P1 ships, that must still be worth having.

Each story is INDEPENDENTLY:
  - buildable   — doesn't need P2 to exist
  - testable    — can be checked on its own
  - demoable    — the owner can open it and use it

Order by importance to the owner, NOT by build order. Build order is the
plan's job; getting these confused is how a spec turns into a task list.
-->

### US1 — <short title> (Priority: P1)

<!-- The journey in plain language. Who, doing what, and what they get. -->

**Why this priority:** <what it buys, and why it outranks the others>

**Independent test:** <how this is checked alone — "open X, do Y, see Z">

**Acceptance scenarios:**

1. **Given** <starting state>, **When** <the owner does this>, **Then** <what
   they see>
2. **Given** <starting state>, **When** <this goes wrong>, **Then** <what they
   see instead>

<!--
Write at least one scenario for the unhappy path. The happy path is the one
that gets built anyway; the failure path is the one that decides whether the
app feels trustworthy or broken.
-->

---

### US2 — <short title> (Priority: P2)

**Why this priority:** <…>

**Independent test:** <…>

**Acceptance scenarios:**

1. **Given** <…>, **When** <…>, **Then** <…>

---

<!-- Add more stories as needed. Three is a common number; one is legitimate. -->

## Interface & experience

<!--
What this should look like and feel like to use. The owner's priority, so it
gets its own section rather than being scattered through scenarios.

DESCRIBE EXPERIENCE AND STATES, NOT CSS. "The empty state explains what to do
next and offers the button that does it" — yes. "16px muted heading, 24px
gap" — no; that is the design doctrine's job and the implementer's.

Naming a shadcn component or an existing page to copy the shape of is fine and
useful. Specifying padding is not.
-->

### Surfaces

<!--
Which screens or areas this touches, and what each is FOR in one line. Say
whether each is new or already exists.
-->

| Surface | New or existing | What the owner does here |
|---|---|---|
| <screen or area> | <new \| existing> | <the one thing it is for> |

### The four states

<!--
MANDATORY, per AGENTS.md §3.9. Every surface above ships all four together —
a surface that only has a populated state is not finished.

The empty state is the one that gets skipped and the one that matters most:
it is what the owner sees on day one, before any data exists.
-->

| State | What the owner sees |
|---|---|
| **Populated** | <the normal case> |
| **Empty** | <what it says, and the action it offers — never a bare "No items"> |
| **Loading** | <skeleton shaped like the real content, or what else> |
| **Error** | <what failed, in plain words, and the next action> |

### Flow

<!--
The path through, step by step, as the owner walks it. Where they start, what
they click, where they land. Name the dead ends.

DELETE if the feature is a single surface with no journey through it.
-->

## Edge cases

<!--
The boundaries, as questions. What happens when there are zero of something?
Ten thousand? When it's offline, half-finished, or the owner does it twice?

An unanswered edge case here is fine — that is what this section is for. An
UNASKED one becomes a bug report later.
-->

- What happens when <boundary condition>?
- How should it behave when <error scenario>?

## Requirements

<!--
Numbered so tasks can cite them. Each is testable — someone can point at the
running app and say met or not met.

Still no implementation. "The system must remember which machine I used last"
is a requirement; "add a last_used_at column" is a plan decision.

Mark genuine unknowns inline:
  FR-00n: System MUST … [NEEDS CLARIFICATION: <the specific unknown>]
-->

### Functional requirements

- **FR-001**: System MUST <capability, in the owner's terms>
- **FR-002**: Owner MUST be able to <interaction>

### Key entities

<!--
The THINGS this feature deals with, conceptually — what they represent and how
they relate. No columns, no types, no table names; the plan turns these into
schema.

DELETE if this feature involves no new concepts.
-->

- **<Entity>**: <what it represents, and what it relates to>

## Success criteria

<!--
How the owner knows this actually worked, measurably, without reference to how
it was built. These outlive the implementation.

Prefer observable outcomes over feelings: "I can tell which machine is mine at
a glance, without opening a detail page" beats "the machines page is clearer".
-->

- **SC-001**: <measurable outcome>
- **SC-002**: <measurable outcome>

## Assumptions

<!--
What was taken as given while writing this, especially defaults chosen because
the description didn't say. Each one is a place this spec could be wrong in a
way nobody notices until it ships.

Include scope boundaries: what this deliberately does NOT cover, and where it
went (Deferred.md / Ideas.md, with the id).
-->

- <assumption, or scope boundary with its register id>

## Owner review

<!--
Filled in at the review gate, before planning starts. Per the review rule, the
owner reads this document and does one of: accepts, asks for changes, or
rejects the direction.

Record the date and the outcome. If stories were re-prioritized or cut during
review, say which and why — that is the most valuable line in the file six
weeks later.
-->

**Reviewed:** <YYYY-MM-DD> — <accepted \| changes requested \| rejected>

<!-- What changed as a result. -->
