# Third-party notices — ai-design-slop

The rule catalogue in `references/refuse-list.md` is derived from two upstream
works, both Apache-2.0. Rules were selected, re-grouped, re-tiered, and
rewritten in this repo's voice; no file is reproduced verbatim, and the schema
and tier model are adaptations rather than copies.

## impeccable

Source of the majority of the container, palette, type, furniture, motion, and
substitution rules, and of three structural ideas this skill and `slop-audit`
adopt: the rule-registry schema, the two-tier surfacing split (adapted here into
confidence tiers, because this family never blocks a build), and the
narrowest-exception suppression ladder.

| | |
|---|---|
| **Original work** | https://github.com/pbakaus/impeccable |
| **Original licence** | Apache License 2.0 |
| **Taken** | ~25 of 32 `category: 'slop'` rule definitions, plus the `craft-floor.md` refuse list |
| **Not taken** | The detector implementation, the command surface, the platform references, and the project doctrine it ships with |

Rules deliberately dropped: `overused-font` (a typography call that belongs to
`DESIGN.md`, not to a generic catalogue), the four `design-system-*` drift rules
(project-relative — see `references/drift.md`), and the model-fingerprint rules
`codex-grid-background`, `gpt-thin-border-wide-shadow`, and
`theater-slop-phrase`, which name specific generations and date quickly.

## anthropics/skills — frontend-design

Source of the three named default clusters in section 3 of the refuse list, and
of the restraint principle behind `announced-restraint`.

| | |
|---|---|
| **Original work** | https://github.com/anthropics/skills/tree/main/skills/frontend-design |
| **Original licence** | Apache License 2.0 |
| **Taken** | The three AI-default clusters it names, restated as rules with ids |
| **Not taken** | The skill itself. It generates an aesthetic per brief, which would compete with `DESIGN.md` and re-decide the look on every screen |

## Changes from the originals

Per Apache-2.0 §4(b), this is the notice that these files are modified:

- Rules were filtered by a single test — would this still be slop in someone
  else's app? — which moved every project-relative rule out of the catalogue and
  into a pointer file.
- Severity was replaced by confidence (`certain` / `judgment` / `advisory`),
  because this family reports and never fixes.
- Enforcement (hooks, per-edit blocking, the detector) was not adopted; the
  audit is report-only.
