# Button

<!-- These are the usage notes an agent reads before using this component.
     Write what the source code cannot say: which variant means what, and
     which combinations are wrong. -->

Source of truth: `apps/web/components/ui/button.tsx`

## Usage

```tsx
<Button variant="default">Connect computer</Button>
```

## Variant → meaning

| Variant | Use for |
|---|---|
| `default` | The one primary action in the current decision |
| `secondary` | A useful alternative that should not compete with the primary action |
| `outline` | Deferral, cancellation, and lower-commitment actions |
| `ghost` | Compact chrome and actions in rows or toolbars |
| `destructive` | Irreversible or session-ending actions, with explicit confirmation |
| `link` | Navigation inside prose; never the primary submit action |

## Notes

- Keep one default button per local decision surface.
- Use a text label for pairing, retry, and disconnect; an icon alone is too ambiguous.
- Loading keeps the action width stable, disables repeat submission, and uses an ellipsis in the label.
- Do not use provider color for actions. Provider color identifies the provider only.
