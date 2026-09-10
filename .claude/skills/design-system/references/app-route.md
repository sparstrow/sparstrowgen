# The optional in-app route

`index.html` is the primary view and works everywhere. An in-app route is a
*second* view that renders the **real, live components** instead of static
depictions — worth building only when that difference earns its ongoing cost.

## When it is worth it

Add it when all of these hold:

- The app is React and already running a dev server.
- You are in **mirror mode** — there are real components to show.
- Someone will actually use it. A design-system route nobody opens is pure
  maintenance.

Skip it for greenfield (nothing live exists yet) and for any non-React stack.

## The cost, stated plainly

The static viewer is *derived* — add a card, rebuild, done. The app route
**cannot be derived**: rendering a live `<Button variant="destructive">` requires
someone to have written that JSX. So the route needs a hand-maintained registry
mapping card ids to render functions, and that registry is a second place that
goes stale.

That is the real tradeoff: live truth, at the price of a file that drifts. Which
is precisely why the static viewer stays primary and this stays optional.

## Shape

One route, one registry, sharing the same `system.json` so nav order and grouping
never diverge between the two views.

```
app/design-system/
├── page.tsx        reads system.json, renders the shell
└── registry.tsx    card id → () => JSX   ← the hand-maintained part
```

The registry is explicit on purpose. A clever auto-registration that globs the
component directory and renders each export with default props produces a wall
of empty buttons — it shows that components *exist*, which nobody needed to
know, rather than what each variant looks like, which is the entire point.

```tsx
// registry.tsx
export const registry: Record<string, () => React.ReactNode> = {
  "buttons-button": () => (
    <Row label="Variants">
      <Button>Primary</Button>
      <Button variant="secondary">Secondary</Button>
      <Button variant="outline">Outline</Button>
      <Button variant="ghost">Ghost</Button>
      <Button variant="destructive">Destructive</Button>
    </Row>
  ),
};
```

Keys match the card `id` the generator produces (the card's path with separators
replaced by `-`), so the same nav metadata drives both views and a card missing
from the registry can be reported rather than silently absent.

## Gate it out of production

This route exists for developers. Ship it to production and you expose internal
component inventory on a public URL, and pay bundle cost for it.

```tsx
// Next.js App Router
export default function Page() {
  if (process.env.NODE_ENV === "production") notFound();
  // …
}
```

A build-time environment check is stronger than a runtime one where the
framework supports it, since it can drop the code entirely rather than merely
refusing to serve it.

## Keeping the two views honest

The static viewer is generated from cards; the route renders live components.
They can disagree — and when they do, **the route is right**, because it is
rendering the real thing. That disagreement is a signal the card has drifted, so
treat it exactly like a `check` finding: fix the card, do not adjust the route to
match it.
