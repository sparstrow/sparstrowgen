# sparstrowgen design system

This is the rendered, agent-readable companion to root [`DESIGN.md`](../DESIGN.md). The doctrine
defines what the interface should feel like; this folder shows the tokens and primitives that make
those rules concrete.

## Source of truth

- Product doctrine: `../DESIGN.md`
- Product intent: `../PRODUCT.md`
- Live tokens: `../apps/web/app/globals.css`
- Live primitives: `../apps/web/components/ui/`
- Mirror manifest and fingerprints: `system.json`

The system is in **mirror mode**. Change real product tokens or primitives at their live source,
then update the relevant card and run the sync and check commands. Do not patch a card until it
looks right while leaving the product source behind.

## Commands

```powershell
node .claude/skills/design-system/scripts/ds.mjs build --root design-system
node .claude/skills/design-system/scripts/ds.mjs check --root design-system
```

The generated `index.html` is a local catalogue. Foundation cards cover color, type, and spacing;
component cards exist only where usage meaning is not obvious from the primitive source.

## Structure

- `tokens/` — mirrored color values and local documentation-only type/spacing aliases
- `guidelines/` — rendered foundation cards
- `components/` — rendered variants plus concise agent usage notes
- `designs/` — clickable feature prototypes after a rendered direction is chosen
- `shots/` — image directions used to choose a composition before prototype work
- `DECISIONS.md` — reusable decisions taken from owner reactions to rendered work
- `CHANGELOG.md` — newest-first changes to the design system
