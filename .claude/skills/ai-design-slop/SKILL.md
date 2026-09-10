---
name: ai-design-slop
description: >-
  The catalogue of AI design slop — the visual tells that mark an interface as
  machine-generated rather than designed. Names each one with an id, what it
  looks like, why it reads as AI, and the direction out. Load it BEFORE writing
  UI so the tells are never introduced, and it is what `slop-audit` scans an
  existing app against. Use whenever a screen "looks AI", "looks like every
  other app", "feels templated", "looks generic", or whenever a page, component,
  or prototype is about to be built or reviewed. Do NOT use it to decide what
  this product should look like — that is DESIGN.md's job, and this file
  deliberately states none of it.
license: Apache-2.0
metadata:
  family: design
  companion-skill: slop-audit
  consumers: frontend-builder, slop-killer
---

# AI design slop

A catalogue, not a procedure. It names the tells; `slop-audit` is what scans for
them, and `DESIGN.md` is what decides the actual design.

This is the `design` family. Later families (`ai-coding-slop`,
`ai-database-slop`) use the same schema and the same tiers, so `slop-audit` and
the `slop-killer` agent work on them unchanged.

## The rule this file turns on

Every candidate rule sorts by one question: **would this still be slop in
someone else's app?**

| | Where it lives |
|---|---|
| **Yes — absolute tell.** Gradient text, kicker above a heading, emoji standing in for an icon | Here, in [references/refuse-list.md](references/refuse-list.md) |
| **No — relative drift.** An untokenised colour, the wrong icon set, a contrast floor, a surface character | `DESIGN.md` and `design-system/`. [references/drift.md](references/drift.md) says where to look — it copies nothing |

**This file states no rule from `DESIGN.md`, and must never start.** That is not
tidiness. `design-system/DECISIONS.md` records the defect: design rules were
duplicated into a conformance skill, so changing the doctrine left the copy
enforcing the retired rules for every agent that loaded it. `DESIGN.md` §12
carries the ban in its own words. A catalogue that hardcodes the doctrine can
never be re-pointed at a new one.

The practical test when adding a rule here: if it mentions a value, a token
name, or a component this project happens to use, it is drift and belongs in
`drift.md` as a pointer instead.

## Rule schema

Every rule, in every family, carries these fields:

| Field | Meaning |
|---|---|
| `id` | Stable kebab-case handle. Findings cite it; suppressions name it |
| `name` | Short human label |
| `tell` | What it looks like — concrete enough to spot without judgment |
| `why` | Why it reads as machine-made. Without this a rule is just taste |
| `direction` | The way out, as a direction, never as code |
| `tier` | `certain` · `judgment` · `advisory` |
| `detect` | `static` (findable in source) or `render` (needs a painted page) |

## Tiers are confidence, not severity

The audit reports and stops; nothing here blocks a build. So the tier says how
sure a finding is, which is what a reader needs to triage it.

| Tier | Meaning | Treatment |
|---|---|---|
| `certain` | Mechanical and unambiguous. If it matched, it is there | Fix it, or say why it stays |
| `judgment` | A real tell that needs a human read of the surface | Look before acting |
| `advisory` | Reported in its own section, never counted as a failure | Opt-in taste |

Tiering exists because of a measured effect worth knowing: a steady stream of
low-confidence nags makes a model *more* conservative, not more careful. Report
`certain` findings prominently, group the rest, and keep `advisory` out of the
count.

## How to use it

**Building (`frontend-builder`).** Read the refuse list once before writing UI.
Do not narrate the checklist or announce compliance — it is a list of reflexes
to not have, and a screen that names its own restraint is its own tell.

**Auditing (`slop-killer`).** The `slop-audit` skill drives it. This file
supplies the rules; that one supplies the passes, triage, and report shape.

**A rule the brief actually earns.** These are the category's defaults, not
bans — one exception, `kicker-above-heading`, is marked as a hard ban and says
so. When `DESIGN.md`, a pinned brief, or the owner has explicitly chosen
something on this list, that choice wins and the finding is a false positive.
Record it the narrow way `slop-audit` describes; never widen a suppression to
make a fix go away.

## Scope boundaries

- **No production code.** This is a catalogue.
- **No design decisions.** What this product looks like is `DESIGN.md`'s, written
  with the owner via `design-brief`. If a screen needs something the doctrine
  lacks, that is a `DESIGN.md` change with sign-off, not a rule added here.
- **No suppressions written from this file.** `slop-audit` owns that ladder.
- **Nothing project-specific added here.** A rule naming this repo's tokens,
  components, or palette has leaked from `drift.md` and belongs back in it.

Provenance and licences: [NOTICE.md](NOTICE.md).
