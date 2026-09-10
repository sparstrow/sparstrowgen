# File conventions

The whole system rests on four filename suffixes. Nothing scans file *contents*
to decide what something is — the suffix is the contract, and the generator
walks directories rather than reading an index you have to keep updated. That
is deliberate: a hand-maintained index is the first thing to go stale.

## The four suffixes

| Suffix | What it is | Rendered as |
|---|---|---|
| `*.card.html` | A small, self-contained preview tile | An isolated iframe in `index.html` |
| `*.prompt.md` | Usage notes for whoever (human or agent) uses the thing | The "Usage notes for Claude" collapsible under its card |
| `*.dc.html` | A full, standalone, clickable prototype | Linked from its card as "Open live ↗" |
| `*.d.ts` / `*.jsx` | Component types and implementation | Not rendered; consumed by the app (greenfield/React only) |

## Directory layout

```
design-system/
├── index.html              GENERATED — never hand-edit, it is overwritten
├── system.json             GENERATED/maintained — the manifest, see below
├── README.md               the masthead: product context, foundations, file index
├── CHANGELOG.md            newest first
├── styles.css              @import only — no values live here
├── tokens/
│   ├── colors.css          the actual custom properties
│   ├── typography.css
│   └── spacing.css         radius, shadow, motion also live here
├── guidelines/             one card per foundation, flat
│   ├── colors-surface.card.html
│   ├── type-scale.card.html
│   ├── spacing-scale.card.html
│   ├── radius.card.html
│   ├── shadows.card.html
│   └── motion.card.html
├── components/
│   └── <group>/            group = a tight family, not one dir per component
│       ├── button.card.html
│       ├── button.prompt.md
│       ├── Button.jsx      greenfield + React only
│       └── Button.d.ts     greenfield + React only
├── lib/
│   └── <app>-data.js       shared seed data — see below, this matters
├── designs/
│   └── <Category>/         becomes a nav group verbatim
│       ├── sales-orders.dc.html
│       ├── sales-orders.card.html
│       └── sales-orders.handoff.md
└── assets/                 logos, icons, fonts
```

## How grouping works

The generator derives the rail from the directory tree, so **folder names are
user-facing**. Name them the way you want them read.

- A card directly inside `guidelines/` groups under **Foundations**.
- A card inside `components/<group>/` groups under **`<group>`**, title-cased.
- A card inside `designs/<Category>/` groups under **`<Category>`** verbatim.

Cards sort alphabetically within a group. If you need a specific reading order,
name for it (`01-…`) only as a last resort — usually it means the group is too
big and should be split.

## What a card must contain

A card is a complete HTML document. Two things are load-bearing:

```html
<title>Button</title>
<meta name="description" content="Button — variants, sizes, and states">
```

The generator reads the card's own `<title>` and `<meta name="description">` for
the heading above it. The card is the single source of truth for how it
presents itself; nothing else has to be kept in sync.

Link the stylesheet relatively (`../../styles.css` from two levels deep) so
tokens resolve when the card is opened directly on its own, not only inside the
viewer.

Because cards render inside sandboxed iframes, a card may ship any `<style>` it
likes — including a global reset — without affecting the page around it. Keep
each one small; it should communicate one idea.

## `system.json` — the manifest

This is what makes maintenance possible rather than aspirational. Without a
recorded fingerprint of each mirrored source there is nothing to diff against,
and "maintaining" the system degrades into regenerating it and hoping.

```jsonc
{
  "name": "Sparstrowgen",
  "mode": "mirror",                    // or "greenfield"
  "theme": { "default": "dark", "toggleAttr": "data-theme" },
  "sources": {
    "tokens": ["packages/ui/styles/globals.css"],
    "components": ["packages/ui/components/ui"]
  },
  "tokens": { "--primary": "oklch(0.205 0 0)" },   // recorded values, diffed by `check`
  "mirrors": [
    {
      "card": "components/buttons/button.card.html",
      "source": "packages/ui/components/ui/button.tsx",
      "fingerprint": "sha256:b643fef14d172b08"
    }
  ]
}
```

Paths inside `system.json` are always forward-slashed, including on Windows —
`check` normalizes both sides before comparing, and a raw `path.relative()`
result would otherwise report a correctly-registered card as unmirrored drift.

## `lib/` seed data — why it is not optional

Every prototype should import the *same* fake company: the same customers, the
same order numbers, the same product names. This is the difference between a set
of prototypes and a believable product.

Concretely, shared seed data means a reviewer can follow one order from the list
view into the detail drawer into the invoice, and it is the same order. Ad-hoc
placeholder data per prototype makes cross-screen review impossible and hides
real problems — column widths that only break on a long customer name, totals
that only misalign at four digits.

Keep it as a plain JS module exporting arrays, so both `.dc.html` prototypes and
`.card.html` cards can use it with a `<script src>`.

## Naming

- Files: lowercase kebab (`type-scale.card.html`, `sales-orders.dc.html`).
- Component implementations: PascalCase (`Button.jsx`), matching the export.
- Design categories: human title case with spaces (`ERP App`, `Sales & Marketing`)
  — they are read directly by users in the rail.

## Generated files

`index.html` is overwritten on every `build`. Never hand-edit it; put the change
in the card, the README, or the viewer shell in the skill's `assets/`.

Whether `index.html` is committed is a project call. Committing it means anyone
can open the system from a fresh clone with no toolchain — usually worth the
diff noise. Gitignoring it keeps history clean but makes the system invisible
until someone runs a build.
