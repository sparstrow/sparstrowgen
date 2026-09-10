# Drift — the project-relative half

A tell is **absolute** when it would still be slop in someone else's app.
Everything else is **drift**: correct-looking work that violates *this*
project's doctrine. Both matter; only the first can be written down here.

**This file names where the answers live. It copies none of them, and must
never start.** `design-system/DECISIONS.md` records what happened the last time
design rules were duplicated into a skill: changing the doctrine left the copy
enforcing the retired rules for every agent that loaded it, and no amount of
re-pointing the doctrine could reach the copy. `DESIGN.md` carries the ban in
its own Do/Don't list.

So a drift check is always the same two steps: **read the source below in the
same turn, then check the surface against what it says today.** Working from a
memory of the rules is the failure this file exists to prevent.

## Where each answer lives

| Drift check | Read | Not from |
|---|---|---|
| Colour, accent behaviour, what may carry meaning | `DESIGN.md` §2 | Memory, or the nearest existing screen |
| Type scale, weights, measure | `DESIGN.md` §3 · `design-system/guidelines/type-scale.card.html` | A framework default |
| Spacing unit and density | `DESIGN.md` §4 | Eyeballing an adjacent component |
| Elevation, depth, radius | `DESIGN.md` §5 · `guidelines/elevation.card.html` · `guidelines/radius.card.html` | Whatever the last card used |
| Icon set, size, stroke, and the semantic map | `DESIGN.md` §6 | Improvisation. A missing icon rule is a gap to raise, not a licence |
| Motion durations, easing, reduced-motion behaviour | `DESIGN.md` §7 · `guidelines/motion.card.html` | A library default |
| Which component to reach for | `DESIGN.md` §8 · `design-system/components/` · the `shadcn` skill and MCP | Composing a new primitive |
| Surface treatments and their worst case | `DESIGN.md` §2 · `guidelines/surfaces.card.html` | The one variant that happened to be active |
| Status and state colour | `guidelines/status-colors.card.html` · `DESIGN.md` §2 | Semantic guesswork |
| Every checkable rule, collected | `DESIGN.md` §11 Named rules | — |
| The project-specific refuse list | `DESIGN.md` §12 Do / Don't | This catalogue, which is deliberately generic |
| Token names and current values | `design-system/system.json` (mirrors the real stylesheet) | Any hardcoded literal, ever |
| What is deliberately not decided | `DESIGN.md` §13 | Filling the vacuum, which makes your guess the standard |

## The one drift rule that is always true

**A literal value where a token belongs is drift, whatever the doctrine says.**
Colour, radius, shadow, spacing, type size. The doctrine defines a theming
contract rather than a fixed palette, so a hardcoded value passes exactly the
one theme it was tested in and silently breaks the rest.

This is the only drift check that needs no lookup — the rule is structural, not
stylistic. What the token is called and what it resolves to still come from
`design-system/system.json`, never from here.

## When the doctrine has no answer

Not a licence to improvise, and not a finding to file. It is a `DESIGN.md`
change needing owner sign-off — see §13 Deliberately undecided, which exists
precisely so an agent asks rather than invents. A quiet one-screen exception is
invisible to every other agent and becomes an inconsistency nobody can trace.

An audit that hits this reports it as an **open doctrine gap**, not as a slop
finding, and names the section that would have to change.
