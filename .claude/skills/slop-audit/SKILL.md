---
name: slop-audit
description: >-
  Audits existing work against a slop catalogue and reports what it finds —
  never fixes, never suppresses, never files. Family-agnostic: it loads
  `ai-design-slop` today, and `ai-coding-slop` / `ai-database-slop` unchanged
  when those exist. Use it on a page, route, component, directory, prototype,
  or the whole app whenever someone asks whether something "looks AI", "feels
  generic", "looks like every other app", or asks for a slop audit, a design
  audit, or a check before shipping a surface. Do NOT use it to fix what it
  finds — hand the report to whoever owns the code.
license: MIT
metadata:
  families: ai-design-slop
  report-only: true
  consumers: slop-killer
---

# Slop audit

Report-only, by construction. The audit reads, scores, and hands back a report;
somebody else decides what to do with it.

That boundary is the whole design. An auditor that also fixes stops being a
second opinion — it becomes the same judgment that wrote the code, checking its
own work.

## Report-only means all four of these

| Never | Why |
|---|---|
| Edit source | The finding is the deliverable. Fixing is a separate decision by whoever owns the surface |
| Write a suppression | Suppressing is how an audit quietly becomes a silencer. The ladder below is advice for the *owner*, executed by them |
| File a `docs/bug/` entry | The report names which findings deserve one; filing is the caller's call |
| Claim a clean surface it did not inspect | Say what was scanned and what was not. An unscanned surface is unknown, not clean |

The `slop-killer` agent declares no `Write` and no `Edit` — but a harness may
grant them anyway (observed 2026-08-19), so the boundary is a rule you keep, not
a wall you are behind. It holds identically when this skill is invoked directly
in a session that has every write tool available.

**The check is `git status --short`, unchanged from before the run.** It is the
one piece of evidence that does not depend on the harness honouring a tool list.

## Flow

### 1. Resolve the family

Default `design` unless the caller names another. Load that family catalogue in
this turn — `.claude/skills/ai-design-slop/` for design: its `SKILL.md` for the
schema and tiers, `references/refuse-list.md` for the absolute rules,
`references/drift.md` for where the project-relative answers live.

**Never audit from a memory of the rules.** A catalogue read last week is a
different catalogue.

### 2. Resolve the target

| Caller said | Scan |
|---|---|
| A file or directory | That subtree |
| A route or page name | Its component tree, following imports one level into shared components |
| "the app", nothing | The route-level pages, then the composed domain components. Say in the report that primitives were not swept |

If the target cannot be resolved to real paths, ask once rather than guessing —
an audit of the wrong subtree reads exactly like a clean one.

### 3. Static pass — always

Read and grep the target for every rule whose `detect` is `static`. This is the
default and needs no server. Cite `file:line` for each hit; a finding that
cannot be located is a hypothesis, and hypotheses do not go in the report.

Alongside the absolute rules, run the one drift check that needs no lookup:
**a literal value where a token belongs** (colour, radius, shadow, spacing, type
size). For any other drift check, read the source `drift.md` names and check
against what it says today.

### 4. Render pass — only when it earns itself

Run it when **both** hold: the target is a live route, and a `render` rule is in
scope (`oversized-h1`, `scattered-entrances`, `monotonous-spacing`,
`uniform-section-shell`), or the caller asked for contrast, overflow, or focus
checks.

Use the browser loop the `frontend-verify` skill already defines rather than
standing up a second harness. Check both modes and more than one surface
treatment — a surface with tint to hide behind is not the honest case.

If the target has no route yet, **say so and skip it**. A component that cannot
be painted has unknown render-tier status, which is a different statement from
passing.

### 5. Triage every finding into exactly one of three

| Outcome | When | What the report says |
|---|---|---|
| **Real** | It is there and it is a tell | The finding, with `file:line` and the direction out |
| **Earned** | The doctrine, a pinned brief, or the owner explicitly chose it | Drop it from the findings, and note under Notes that the rule matched but was earned, naming the evidence |
| **Unsure** | You cannot tell without knowing intent | Report it under Open questions with the one question that would settle it |

The bar for **earned** is evidence you can name: a rule in `DESIGN.md`, a
decision in `design-system/DECISIONS.md`, an approved prototype handoff, or
something the owner said in this session. "It looks deliberate" is not evidence.
Write "the owner confirmed" only when they actually did.

### 6. Report

Format in [references/report-format.md](references/report-format.md). Hand it
back as text and stop.

## The suppression ladder — advice, not an action

The audit never writes a suppression. It **recommends** one at the narrowest
rung that covers the case, so the owner can act:

| Rung | Scope | Who may choose it |
|---|---|---|
| Value-scoped | One rule, one value | Recommendable by the audit |
| File-scoped | One rule, one file | Recommendable by the audit |
| Rule-off | One rule, whole project | Owner only — ask |
| File-ignored | Every rule, one file, including rules not yet written | Owner only, and only for a fixture, a generated artifact, or a deliberate slop demo |

**Never recommend a suppression to skip a fix.** A suppression is for a finding
that is wrong or earned; a finding that is right and inconvenient gets fixed or
gets accepted out loud.

## Where findings should go afterwards

The report names destinations; it does not write to them.

| Finding | Destination |
|---|---|
| A user-visible defect in the running app | `docs/bug/`, per AGENTS.md §5 — flagged in the report as bug-worthy |
| A tell that is real but cosmetic | Stays in the report; fixed by whoever owns the surface |
| A doctrine gap (§13 territory) | An owner decision, not a bug. Name the `DESIGN.md` section that would change |
| A rule that fired wrongly more than once | Worth a catalogue fix — say which rule and why |

## Adding a family later

Nothing above names a design concept. A new family drops in by supplying a
catalogue with the same schema and tiers; the only per-family variation is which
`detect` modes exist — a coding or database family has no render pass, so step 4
is simply skipped and the report says so.

## Scope boundaries

- **No fixes, no suppressions, no bug files.** See the table at the top.
- **No design decisions.** What the product should look like is `DESIGN.md`'s.
- **No inventing rules.** If a surface is bad in a way the catalogue does not
  name, report it under Notes as an uncatalogued observation and propose the
  rule — do not score it as a finding.
- **No unscanned claim.** Coverage is stated, always.
