# Mirror mode — documenting an app that already has components

In mirror mode the app's code is the source of truth and the design system is a
lens on it. The deliverable is **understanding**, not implementation: cards that
show what exists, and usage notes that carry the judgment the source cannot.

## Finding the sources

Before `init`, locate three things:

1. **The token stylesheet** — the file defining CSS custom properties. Common
   locations: `**/globals.css`, `**/theme.css`, `**/tokens.css`, or a Tailwind
   `@theme` block. Grep for `--background:` or `--primary:` to find it fast.
2. **The component directory** — `components/ui/`, `src/components/`, or a
   dedicated workspace package. A `components.json` points at it directly.
3. **Any written design doc** — `DESIGN.md`, brand guidelines, a Figma export.
   This is where the *prose* for `README.md` comes from; without it you are
   inferring intent from CSS, which produces a description of what the code does
   rather than what it means.

Pass 1 and 2 to `init` as `--token-source` and `--component-source`.

## Tokens

Do not retype the values. Either `@import` the real stylesheet, or copy the
values with a comment naming the file they came from and the date. Copying
without attribution is how a token file becomes a second, quietly-wrong source.

```css
/* tokens/colors.css
   Mirrored from packages/ui/styles/globals.css.
   Do not edit here — edit the source and run `ds.mjs sync`. */
```

`init` records every token value into `system.json`, so `check` can tell you
when the real stylesheet moves out from under you.

## Cards

A mirror-mode card is **hand-written HTML that reproduces the component's
appearance using the real tokens.** It is not the real React component — a
static page cannot mount one without a bundler, and adding a bundler to the
design system is how a zero-dependency tool becomes a build system.

This is an honest tradeoff and worth naming: the card is a *depiction*, and a
depiction can drift. Three things keep that in check:

1. **Fingerprinting.** Register every card with `--source`. When the real
   component changes, `check` tells you the card may now be wrong. The card can
   drift, but not *silently*, which is the property that matters.
2. **Tokens, never literals.** Write `var(--primary)`, never `#c8794a`. A card
   built from tokens follows the real theme automatically, so the entire class
   of colour drift disappears.
3. **Read the source before writing the card.** Not the prop types — the actual
   variant definitions (`cva` maps, `class-variance-authority` configs, styled
   variants). That is where the real list of variants lives, and it is routinely
   longer than what anyone remembers.

For a React app that genuinely needs live components rather than depictions,
that is what the optional in-app route is for. See
[app-route.md](app-route.md).

## Usage notes are the actual deliverable

In greenfield mode the implementation is the artifact. In mirror mode the
implementation already exists — so the `.prompt.md` *is* what you are producing.
Spend the effort there.

Write what the code cannot say:

```markdown
## Variant → meaning

| Variant | Use for |
|---|---|
| `ok` | Shipped, Paid, Certified, Released — a terminal good state |
| `warn` | Low stock, approaching due date — needs attention, not yet wrong |
| `bad` | Overdue, Blocked, negative quantity — actually wrong |
| `neutral` | Draft, Unset — no signal intended |
```

That table cannot be derived from `variant: "ok" | "warn" | "bad"`. It is the
difference between an agent picking a colour and an agent picking a *meaning*.

Also capture:
- Combinations that are wrong (`size="pill"` only for nav counts, never table cells).
- The gotcha someone already hit.
- Real usage copied from the codebase, not invented.

## The drift workflow

```bash
node ds.mjs check --root design-system
```

Findings and what each means:

| Finding | What happened | What to do |
|---|---|---|
| `token-changed` | A value differs between the system and the real stylesheet | Update `tokens/` and any card that hardcoded it, then `sync` |
| `token-added` | The app gained a token the system does not document | Add it to `tokens/` and usually a foundation card, then `sync` |
| `token-removed` | The system documents a token the app deleted | Remove it and fix cards referencing it, then `sync` |
| `component-changed` | A mirrored source changed since its card was written | Re-read the source, update card + usage notes, then `sync` |
| `dangling-mirror` | A card documents a file that no longer exists | Delete the card or repoint it |
| `unmirrored-card` | A component card has no registered source | Re-add with `--source`, or accept it is unwatched and say why |
| `prototype-uncarded` | A `.dc.html` has no preview card | Add one, or it never appears in the index |

**`sync` last, always.** It re-records fingerprints so drift stops being
reported — which is correct *after* you have updated the cards, and a way of
silencing a true warning if run before. If you catch yourself running `sync` to
make `check` quiet, that is the moment the system starts lying.

## Wiring `check` into CI

`check` exits 1 on drift, so it needs no wrapper:

```yaml
- name: Design system drift
  run: node .claude/skills/design-system/scripts/ds.mjs check --root design-system
```

Consider starting it non-blocking (`continue-on-error: true`) for the first few
weeks. A brand-new system tends to surface a burst of legitimate findings, and a
gate that fails on day one gets disabled rather than fixed.
