# Greenfield mode — starting a design system from nothing

Here the design system *is* the source of truth. Components are authored in it
and graduate into app code later. The risk is the opposite of mirror mode: not
drift, but arbitrariness — a pile of values nobody can defend six weeks later.

## Decide the character before the values

Write one sentence naming what this product feels like, and let every later
choice answer to it. "A quiet, dense control plane for engineers" and "a warm,
spacious tool for small-business owners" produce different scales, different
contrast, different radii. Without that sentence you get defaults, and defaults
from a model are recognisably generic — the flat purple-gradient SaaS look that
signals nobody made a decision.

Put the sentence at the top of `README.md`. It is the thing to re-read when a
later choice feels arbitrary.

## Tokens — a defensible starting point

These are starting points to *depart from deliberately*, not defaults to accept.

**Color.** Work in OKLCH — it is perceptually uniform, so equal lightness steps
look equal, which HSL does not give you. Define:
- A surface stack, deepest to lightest, 3–5 steps (`--sidebar` → `--background`
  → `--card` → `--muted`). This carries hierarchy; it is why you rarely need
  shadows.
- Foreground hierarchy, 3 steps (`--foreground`, `--muted-foreground`,
  `--neutral-foreground` for placeholder/disabled).
- One brand/primary plus a hover state.
- Semantic status colours. Four is usually right: ok, info, warning, danger,
  plus a neutral. Hold chroma roughly constant across them (`~0.13–0.16`) and
  vary only hue, so no single status shouts louder than the others by accident.
- `--border`, `--input`, `--ring`.

Avoid pure `#000` and `#fff` for surfaces — tinted near-blacks and near-whites
read as considered, and pure black clips on OLED.

**Type.** One family for UI, one mono for codes, IDs, and figures. A 5–6 step
scale is enough; more means nobody can tell two of them apart:

| Step | Typical | Use |
|---|---|---|
| `--text-xl` | 18px / 700 | Page title |
| `--text-lg` | 15px / 600 | Section heading |
| `--text-body` | 13px / 400 | Body, primary content |
| `--text-base` | 12.5px / 400 | Secondary content, meta |
| `--text-sm` | 11.5px / 600 | Column headers, labels |
| `--text-xs` | 11px / 600 | Micro labels, badges |

Dense tools sit smaller than marketing sites; the numbers above suit an
information-dense app. Tighten letter-spacing slightly on headings (`-0.02em`).
Set a minimum body size and hold it — the pressure to shrink text to fit is
constant and always wrong.

Use `font-variant-numeric: tabular-nums` anywhere numbers are compared down a
column. Without it, right-aligned currency in a table is subtly ragged.

**Spacing.** A 4px base grid, exposed as `--space-1` … `--space-12`. Name the
common uses so the scale gets applied consistently rather than picked at random:
`4` icon gaps · `8` icon+label · `12` inline · `14` form fields · `20` list rows
· `24` section padding.

**Radius.** 4–5 steps, each with a stated use — `xs` chips, `sm` inputs and
buttons, `md` cards and panels, `lg` modals, `full` pills and avatars. Radius is
where inconsistency shows fastest because mismatched corners sit adjacent.

**Shadows.** Three at most: a base, a hover lift, and a large one for
drawers/modals. If the surface stack is doing its job, most surfaces need none.

**Motion.** Two or three named keyframes, not a library. A fade-with-rise for
section entrance, a pop for dropdowns, a slide for toasts. Durations 150–200ms,
`ease`. Name them and reuse them; ad-hoc transitions are how an interface
develops a stutter. Honour `prefers-reduced-motion`.

## Component implementation format

Depends on the target app:

**React target** — author `.jsx` + `.d.ts` alongside the card. The component is
real and portable, and `.d.ts` gives an agent the signature. Mount them under a
single global so a plain `.dc.html` prototype can consume them without a build:

```html
<script src="../components/buttons/Button.jsx" type="text/babel"></script>
<script>const { Button } = window.AppDesignSystem;</script>
```

**Unknown or non-React target** — HTML + CSS only. The card is the component:
markup plus token-driven classes, copy-pasteable into any framework. You lose
the typed API surface, and gain portability to Vue, Svelte, or server-rendered
templates. Choose this when the stack is genuinely undecided; converting
HTML+CSS to a framework component later is cheap, whereas unpicking React from
a design system is not.

Either way the card and `.prompt.md` are mandatory. The implementation is the
part that varies.

## Graduating into the app

When the app is built for real, the design system's components become the
reference, not the runtime. Copy the values, not the files: the design system
runs on a CDN-grade setup deliberately, and shipping that into production
inherits choices made for speed of iteration.

Keep the design system afterwards — do not delete it. Once real components
exist, **switch the system to mirror mode**: edit `mode` in `system.json`,
register each card against its now-real source with `--source`, and `check`
starts protecting you from the drift that begins the moment two copies exist.
That transition is the natural end of greenfield mode, and skipping it is how
the two copies quietly diverge.
