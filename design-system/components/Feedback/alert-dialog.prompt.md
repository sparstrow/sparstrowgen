# Alert dialog

<!-- These are the usage notes an agent reads before using this component.
     Write what the source code cannot say: which variant means what, and
     which combinations are wrong. -->

Source of truth: `apps/web/components/ui/alert-dialog.tsx`

## Usage

```tsx
<AlertDialog>
  <AlertDialogContent>…</AlertDialogContent>
</AlertDialog>
```

## Variant → meaning

| Variant | Use for |
|---|---|
| `default` | Confirm a consequential action with enough room to state the effect |
| `sm` | Short binary confirmations without supporting detail |

## Notes

- Use for disconnect because active work may end and the computer must be paired again later.
- The title names the action. The description names what stops and what remains safe.
- Put the safe alternative first in reading order and the destructive action last.
- Do not use a dialog for the pairing journey; it is progressive work and needs durable space.
