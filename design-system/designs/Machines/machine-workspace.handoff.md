# Machine workspace — handoff

| | |
|---|---|
| **Prototype** | `machine-workspace.dc.html` (serve through the design-system catalogue; a bare file has no token imports) |
| **Provenance** | [`docs/specs/2026-09-12-first-usable-release.md`](../../../docs/specs/2026-09-12-first-usable-release.md), US2; owner selected shots A + B on 2026-09-13 |
| **Mode** | build |
| **Status** | draft — awaiting owner review before production wiring |
| **Design system** | shadcn/ui with Base UI (`base-nova`), semantic tokens mirrored from `apps/web/app/globals.css`, Lucide icons, existing provider marks |

## What this is

A dedicated Machines destination for the computers connected to the signed-in account. The list
opens a focused profile showing connection status and agent providers. The same destination owns
first-time pairing and disconnecting; Chat remains a separate daily work surface.

## Component mapping

| Prototype element | Production component | Status |
|---|---|---|
| Persistent Chat / Machines navigation | shadcn `Sidebar`, `SidebarMenu`, `SidebarMenuButton`, `SidebarInset` | **registry component; not installed** — add through `pnpm exec shadcn add sidebar` |
| Primary, outline, ghost and destructive actions | existing shadcn `Button` | installed |
| “This computer” label | existing shadcn `Badge` | installed |
| Machine and provider rows | shadcn `Table` | **registry component; not installed** |
| Machines / computer path | shadcn `Breadcrumb` | **registry component; not installed** |
| No machines | shadcn `Empty` | **registry component; not installed** |
| Loading rows | existing shadcn `Skeleton` | installed |
| Disconnect confirmation | existing shadcn `AlertDialog` | installed |
| Section boundaries | existing shadcn `Separator` | installed |
| Transient confirmation | existing Sonner toaster | installed |
| Control and state icons | Lucide | installed; keep text with every non-universal meaning |
| Claude, Codex and agy marks | existing `ProviderIcon` | installed business component; do not redraw |
| “Prototype only” state controls | prototype scaffolding | **do not port** |

## Token usage

Only existing semantic tokens are used: background, foreground, card, popover, muted, secondary,
accent, border, input, ring, primary and destructive, plus the three existing provider identity
tokens. No success token is invented; online and available states use an icon plus a label.

## States

| Surface | States represented |
|---|---|
| Machines index | populated, empty, loading, error |
| Machine profile | online, offline, disconnected after confirmation |
| Pairing | introduction, looking, unanswered, approval, mismatch, success |
| Recovery | list retry, pairing retry, installer help, pair again |

The bottom review control exposes the four index states and pairing mismatch directly. It is not
part of the product.

## Data contract

| Data / behaviour | Source required from backend |
|---|---|
| Machine id, display name, “this computer”, account owner | account-scoped machine record |
| Online/offline and last seen | daemon connection presence plus persisted last-seen time |
| Active work summary | account-scoped active work for that machine |
| Provider name and availability reason | daemon-reported provider capability |
| Pairing request identity and expiry | one-time pairing request/claim |
| Approval and mismatch outcome | server-issued credential bound to the approved account and machine |
| Disconnect | credential revocation; future attempts refused until approved again |

Realtime presence changes should patch or invalidate the TanStack Query cache. Provider snapshots
arrive as whole machine capability updates. Setup waits on a bounded request status;
successful pairing persists across browser, server and computer restarts. Runtime versions, logs,
restart controls and cost data are deliberately absent because US2 cannot serve them.

## Interactions

- Machine row → focused profile. Keep.
- Add computer → progressive pairing page. Keep.
- The first lookup times out to the honest “has not answered yet” state. The retry simulates a
  valid request. Both timings are fake; keep the state sequence, not the timers.
- Approval → success → profile. Simulated locally; wire to the pairing request contract.
- Pairing mismatch connects nothing and offers a safe restart. Keep.
- Installer action only shows a prototype toast. Wire it to the signed Windows installer later.
- Disconnect opens an `AlertDialog`, returns focus on cancel/Escape, and locally changes the
  profile after confirmation. Wire to revocation and wait for the server response before changing
  the Query cache.
- Chat and account controls only explain that they are outside the prototype. Do not port those
  handlers.

## Invented

These details need owner or implementation confirmation:

- A disconnected machine record remains visible and offers **Pair again**.
- Default ordering puts this computer first, then most recently seen.
- `DESKTOP-RIVER`, `FINANCE-LAPTOP`, the account display and all exact copy are seed content.
- The review is desktop-first at a minimum width of 780px; the narrow-screen pattern is undecided.

## Open questions

1. After disconnect, should the machine remain in the list as a record or disappear immediately?
2. Should a newly connected machine always appear first, or should the list remain last-seen order?

## Not included

US3 update/version work; rename; logs; restart/stop; costs; provider/model configuration; registered
folders; the separate Connection facts section; mobile navigation; Chat redesign; backend
implementation.

## Verification

**Tested 2026-09-13:** `http://localhost:4321/designs/Machines/machine-workspace.dc.html`
against this handoff, US2 and root `DESIGN.md`. The complete browser pass after the final fix found
no further issues and no console warnings or errors.

### Checklist

- [x] Populated list shows both realistic machines, full names, status, providers, activity and last seen
- [x] Empty, loading and error states; error explains safety and retry returns to the list
- [x] Route-based list → online profile → breadcrumb return
- [x] Offline profile changes provider availability to “Waiting for computer”
- [x] Provider marks and availability render at the intended scale; the profile ends after providers
- [x] Disconnect confirmation names the selected computer, takes initial focus, closes with Escape,
  restores focus on cancel, and shows the disconnected state after confirmation
- [x] Pairing introduction → unanswered recovery → retry → approval → mismatch safety
- [x] Pairing approval → success → machine profile
- [x] Prototype shortcuts, installer explanation, Chat explanation and account explanation respond
- [x] List, profile and mismatch visually inspected at the normal desktop viewport
- [x] Semantic token scan found no literal hex/RGB/HSL colors; console remained clean

### Found & fixed

- Machine names truncated in the index at the normal review width — root cause: the fixed secondary
  columns consumed the first column's available width. The grid now protects the name column and
  tightens secondary metadata columns.
- Disconnecting `FINANCE-LAPTOP` named `DESKTOP-RIVER` in the confirmation and toast — root cause:
  prototype copy was hard-coded instead of resolving the selected machine. Both now use selection.

### Found & not fixed

- None.

### Environment caveats

- The automation API waits for page settling after a click, so it observes the result after the
  brief “Looking…” and retry-loading transitions rather than those timed frames. The same loading
  geometry was exercised directly through the review state; the synchronous render-before-timer
  path was inspected in the prototype source.
- Pairing, installer download and disconnect are deliberately local simulations; no backend or
  network request exists at this stage.

### Feedback verification — 2026-09-13

After feedback item 1, both online and offline profiles were rechecked in the browser. Registered
folders and the separate Connection section are absent; provider availability, breadcrumb,
disconnect, post-disconnect recovery and list navigation remain intact. The full list and pairing
state pass remained green, and the console remained free of warnings and errors.
