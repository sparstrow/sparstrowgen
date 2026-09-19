# Drift — the project-relative half

A tell is **absolute** when it would still be slop in someone else's app.
Everything else is **drift**: correct-looking work that violates *this*
project's design system. Both matter; only the first can be written down here.

**This file names where the answers live. It copies none of them, and must
never start.** Rules duplicated into a skill keep enforcing themselves after the
design system changes, for every agent that loaded the copy, and re-pointing the
system cannot reach the copy.

So a drift check is always the same two steps: **read the source below in the
same turn, then check the surface against what it says today.** Working from a
memory of the rules is the failure this file exists to prevent.

The design system is the Claude artifact linked from `CLAUDE.md` (§4, Design).
Read `project/README.md` first, then the component's own README, then
`project/tokens.json` for token names, values and usage notes.

## Where each answer lives

| Drift check | Read | Not from |
|---|---|---|
| Colour, accent behaviour, what may carry meaning | README, *Visual foundations* · `tokens.json` colour usage notes | Memory, or the nearest existing screen |
| Type scale, weights, measure | README, *Visual foundations* · `tokens.json` type styles | A framework default |
| Spacing unit and density | README, *Visual foundations* · `tokens.json` spacing | Eyeballing an adjacent component |
| Elevation, depth, radius | README, *Visual foundations* · `tokens.json` radius and shadow | Whatever the last card used |
| Icon set, size, stroke, and the semantic map | README, *Iconography* | Improvisation. A missing icon rule is a gap to raise, not a licence |
| Motion durations, easing, reduced-motion behaviour | README, *Visual foundations* · `tokens.json` duration and easing | A library default |
| Which component to reach for | README, *Components* · the component's own README · the `shadcn` skill and MCP | Composing a new primitive |
| Surface treatments and their worst case | README, *Visual foundations* (appearance) · `tokens.json` themes | The one variant that happened to be active |
| Contrast and known misses | README, *Accessibility* | Assuming a token pair passes |
| Voice, casing, empty and error copy | README, *Content fundamentals* | Generic product copy |
| Token names and current values | `tokens.json` | Any hardcoded literal, ever |

## The one drift rule that is always true

**A literal value where a token belongs is drift, whatever the system says.**
Colour, radius, shadow, spacing, type size. The system defines a theming
contract rather than a fixed palette, so a hardcoded value passes exactly the
one theme it was tested in and silently breaks the rest.

This is the only drift check that needs no lookup — the rule is structural, not
stylistic. What the token is called and what it resolves to still come from
`tokens.json`, never from here.

## When the design system has no answer

Not a licence to improvise, and not a finding to file. It is a design system
change needing the owner's sign-off. A quiet one-screen exception is invisible
to every other agent and becomes an inconsistency nobody can trace.

An audit that hits this reports it as an **open design system gap**, not as a
slop finding, and names the README section or token that would have to change.
