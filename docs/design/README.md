# Design work in progress

The design system itself lives in Claude (linked from `CLAUDE.md`, §4). This folder holds only the
throwaway-to-durable work that happens before a screen is built:

- `shots/<YYYY-MM-DD>-<slug>/` — image directions from `design-shots`, with a `README.md` saying why
  one was chosen. The reasons matter more than the winner.
- `prototypes/<Category>/` — clickable `<name>.dc.html` files and their `<name>.handoff.md` from
  `interactive-prototype`, plus a shared `seed-data.js` and a `tokens.css` copied from
  `apps/web/app/globals.css` and `themes.css`.

Nothing here is the source of truth for what the product looks like. Once a design ships, the running
app and the design system are.
