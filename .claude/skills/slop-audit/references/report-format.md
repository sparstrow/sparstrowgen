# Report format

One report per audit, returned as text. Findings first, coverage always, no
edits anywhere.

## Header

| | |
|---|---|
| **Target** | What was audited, as real paths |
| **Family** | `design` (catalogue version/date read) |
| **Passes** | `static` · `static + render` · and which viewports and modes if rendered |
| **Not scanned** | What was in the neighbourhood and deliberately skipped, and why |

The **Not scanned** row is not optional. A report without it reads as
whole-surface coverage, which is the one lie an audit must never tell.

## Findings

Grouped by tier, most confident first. `advisory` sits in its own section and is
excluded from every count.

Each finding is one row:

| Field | Content |
|---|---|
| `id` | The catalogue rule id, so it can be looked up and suppressed by name |
| Location | `path/to/file.tsx:42` — must open to the thing described |
| Found | What is actually there, concretely. Not the rule restated |
| Why | Why it reads as machine-made, from the rule |
| Direction | The way out, as a direction. Never code, never a diff |

Layout:

```
### Certain (n)

| id | location | found | why | direction |
|---|---|---|---|---|

### Judgment (n)

| id | location | found | why | direction |
|---|---|---|---|---|

### Advisory (not counted)

| id | location | found |
|---|---|---|
```

A tier with nothing in it is written as `### Certain — none` rather than
dropped, so a reader can tell "checked and clean" from "not checked".

## Open questions

Findings that could not be triaged without knowing intent. One line each, each
ending in the single question that settles it. Empty is fine and common.

## Notes

Three kinds, and nothing else:

| Kind | Example |
|---|---|
| **Earned** | A rule matched but the doctrine or the owner chose it. Name the evidence |
| **Uncatalogued** | A real problem the catalogue does not name, with the rule it suggests |
| **Doctrine gap** | The surface needed something `DESIGN.md` does not decide. Name the section |

## Destinations

A short closing table, when any apply. Recommendations only.

| Finding | Suggested destination |
|---|---|
| User-visible defect | `docs/bug/` — the caller files it |
| Doctrine gap | Owner decision on the named `DESIGN.md` section |
| Wrong finding | Value- or file-scoped suppression, at the narrowest rung |

## What a report must never contain

- A patch, a diff, or rewritten code.
- A suppression that was written rather than recommended.
- A score, grade, or percentage. Counts per tier are enough; a single number
  invites gaming the number instead of fixing the surface.
- "Looks good" about anything that was not scanned.

## Worked shape

```
**Target** packages/views/chat/ (11 files)
**Family** design · **Passes** static · **Not scanned** render-tier rules
(no route paints this subtree in isolation)

### Certain (1)

| id | location | found | why | direction |
|---|---|---|---|---|
| emoji-as-icon | board/column-header.tsx:31 | A unicode glyph used as the collapse affordance | Icons are drawn; a glyph matches no stroke or weight and renders differently per platform | Use the icon set the doctrine names |

### Judgment — none
### Advisory (not counted) — none

**Notes** — uncatalogued: three sibling components each define their own row
height. Not a slop tell; suggests a doctrine gap on density.
```
