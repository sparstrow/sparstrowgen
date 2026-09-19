# App shell — handoff

| | |
|---|---|
| **Prototype** | `app-shell.dc.html` (open it from disk; it links `../tokens.css`, `../icons.js` and `../seed-data.js`) |
| **Provenance** | The owner's request, "start the shell restyle like the Claude Design app" ([`Later.md`](../../../Later.md) L-30), and his choice of direction A on 2026-09-19 (D-042). No spec: this restyles what exists. |
| **Mode** | build (direction A chosen from three rendered shells, `docs/design/shots/2026-09-19-app-shell/`) |
| **Status** | draft — waiting for the owner to react to the prototype |
| **Design system** | the Claude artifact linked from `CLAUDE.md` |

## What this is

The frame around every screen: an icon rail, a pane holding the section's own list or navigation, and one
56px header line carrying the page title, a few page controls and the computer's live status. It covers Chat,
Machines and Settings only. The kit's other entries (agents, schedules, pipelines) appear when their features do.

## Component mapping

| Prototype element | Use | Notes |
|---|---|---|
| Icon rail, hover-expands as an overlay | **NEW `Rail`** in a new `components/shell/` | The shadcn `Sidebar` is not reused: its icon mode pushes the content instead of overlaying it, and its `--sidebar-*` tokens are never defined in this app, so its hover and active fills paint nothing (README, "Live gaps"). |
| Pane header, list, search | **NEW `Pane`** wrapping existing content | Chat: the existing `ConversationList` (drop its own title row and "New" button in favour of the pane header). Settings: the nav from `settings-shell.tsx`. Machines: rows built from the existing machine list. |
| 56px header | **NEW `AppHeader`** | Title, optional second line, right-hand slot. |
| Computer status in the header | existing `Status` (`success`, `progress`, `neutral`) | The word is always shown; "Online" alone at narrow widths. |
| Provider strip, composer, transcript, provider chips | existing `ProviderStrip`, `Composer`, `MessageList` | Unchanged; only their surroundings move. |
| Machine rows and profile | existing `Status`, `Button` | Profile content is unchanged. |
| Settings rows and cards | existing `SettingsRow`, `SettingsCard` | |
| Buttons, chips, switch | existing `Button`, `Switch` | The Mode, Surface and Accent chips are hand-built in the app today and stay so (the design system lists a segmented control as missing). |
| Narrow drawer | **NEW** (`Sheet` is not in the design system yet) | Add to the system first if it is built as a sheet, or hand-build it and record the drift. |

## Token usage

Only existing tokens are consumed (`background`, `foreground`, `muted`, `muted-foreground`, `accent`, `border`,
`input`, `ring`, `primary`, `mine`, `success*`, `info*`, `warning*`, `destructive*`, the provider colours). The pane
background is `color-mix(muted 40%, background)`, which is not a token; a `--pane` token is not needed unless the
owner wants the value tuned. **No shadow**: the expanded rail is separated by its border only, because the app has no
shadow token (D-039).

| Needed | Exists? | Action |
|---|---|---|
| `--sidebar`, `--sidebar-accent`, `--sidebar-border` | **no** (never defined in the app) | Not needed: the shell uses `background`, `accent` and `border`. Decide whether to delete the unused shadcn `Sidebar` component or keep it. |
| Radius steps | yes (`--radius` and its derived steps) | The prototype re-derives `--r-sm/md/lg/xl` from `--radius` as `@theme inline` does. |

## States

| State | Reachable in prototype | Notes |
|---|---|---|
| Populated | yes (`?state=populated`) | 8 conversations, 2 archived, 2 computers |
| Empty | yes (`?state=empty`) | Chat: no conversations. Machines: no computers. Copy is a guess. |
| Loading | yes (`?state=loading`) | Skeleton rows in the pane, skeleton transcript. Settings has no loading state. |
| Error | yes (`?state=error`) | "Try again" returns to populated. |
| Computer offline / reconnecting | yes (`?computer=offline`, `reconnecting`) | Header status, muted provider strip, amber "cannot send" notice, disabled send. |
| Narrow (under 768px) | yes (`?width=narrow`, or a narrow window) | Rail and pane become a drawer behind a menu button. |
| Dark | yes (`?theme=dark`) | |

## Data contract

The shell adds almost no data. Checked against [`docs/Capabilities.md`](../../../Capabilities.md).

| Field | Source | Deliverable? |
|---|---|---|
| Section title, page title | route and the selected item | yes |
| Conversation title, folder, tokens, spend | already shown in today's Chat header | yes (existing) |
| Conversation list, search, archived | already loaded by `ConversationList` | yes (existing) |
| Computers, online state, version, providers | `GET /api/machines`, kept fresh by realtime events | yes (existing) |
| Header status "computer online / offline" | the machines query on every page, not only Chat | yes, but **lifted**: today `daemonOnline` lives inside `ChatSurface` only |
| Header status "Reconnecting…" | the browser's own realtime socket state (`api.ts` retries on close) | **partly**: the browser-to-server connection state exists client-side; the computer-to-server "reconnecting" does not exist as a state (a computer is online or not). See the open question. |
| Account email in the rail footer | session | yes (existing `AccountMenu`) |

Nothing here streams. What must survive a refresh: the selected conversation, computer and settings page are in the
URL today (`/`, `/machines/[id]`, `/settings/...`); the prototype keeps them in memory.

## Interactions

Real behaviour, keep: the rail's section switch and hover or focus expansion (overlay, no layout shift); the pane's
selection; conversation search across title and folder including the archive; the Archived toggle; New conversation
adds an untitled row and selects it; the states; the drawer (menu button, backdrop or Escape closes, choosing a section
keeps it open and choosing a page closes it).

Faked, do not port: the send button, Add computer, Disconnect, the folder button, the Rendered/Raw toggle, the
Mode/Surface/Accent chips, Change password and Check now all show a note instead of acting; the transcripts and
counts are seed data.

## Invented

Everything here was decided by the prototype and approved by nobody.

- **The collapsed rail's top mark is a lowercase "s"** in plain type. The design system says the wordmark is plain
  type and there is no logo, so the kit's robot icon was not used; the full wordmark shows when the rail expands.
- **No shadow** on the expanded rail (the kit uses one).
- **The narrow drawer:** tabs for the three sections above the pane, at under 768px. The kit does not show a narrow layout.
- **Pane width 212px, rail 60px expanding to 224px, header 56px**, taken from the kit, not re-derived from the app.
- **The chat header keeps today's contents** (title, editable folder, Rendered/Raw, tokens and spend) in the 56px
  line, with the live status added on the right. The kit's header has only a title and a dot, so this is my merge.
- **Machines becomes a pane and a profile side by side**, instead of a list page that opens a profile page.
- **Settings pages and their copy** in the prototype are approximations of the real ones.
- **Copy**: "No computer connected", "Reconnecting to SRIHARI-DESKTOP…", "Machines could not be loaded", and the empty-pane texts.
- **Search** is in the Chat pane only.
- **The status shows the first computer** when there are two.

## Open questions

- **OQ1:** what does the live status mean? I rendered it as the computer's connection to the server. If the browser's own
  connection dropping should also show, that is a second condition and needs its own words.
- **OQ2:** with two computers (G-37, L-22) which one does the header name? The prototype names the first; this is a guess.
- **OQ3:** does `/machines` open the first computer's profile, or a page with nothing selected? The prototype selects the first.
- **OQ4:** the rail expands on hover, which does nothing on a touch screen, and labels only show on hover. Is that acceptable
  on desktop, or should the labels always show?
- **OQ5:** the Chat pane's "New conversation" button is a plus in the header. The kit's convention is a header action; the
  app today has a labelled button in the conversation list.

## Not included

- The kit's theme toggle in the header: appearance is per account and changes in Settings.
- Any destination the kit has and this app does not (dashboard, agents, projects, task board, messages, runs, pipelines,
  schedule, memory, terminals).
- The account menu's contents (the existing `AccountMenu` is reused).
- Keyboard shortcuts.

## Verification

Run 2026-09-19 with the frontend-verify loop, in the Browser pane against a static server, at 1280 wide and 390 wide.

| Check | Result |
|---|---|
| Wide geometry: rail 60px, pane 212px, header 56px, pane header 56px | pass |
| Rail expands to 224px on hover as an overlay: the rail wrapper stays 60px and the pane does not move | pass |
| Section switch (Chat, Machines, Settings) sets the pane, header title and the current item | pass |
| Chat: select a conversation, transcript follows; untitled conversation; New conversation adds and selects a row | pass |
| Search filters by title and folder, the no-match message shows, focus stays in the box (wide and in the drawer) | pass, after a fix (the search id was duplicated between the pane and the drawer) |
| Archived toggle opens and closes | pass |
| Every seeded conversation has a transcript | pass, after a fix (six showed "Nothing said yet") |
| States: populated, empty, loading, error, for Chat and for Machines; "Try again" recovers | pass |
| Computer online, offline, reconnecting: header status, provider strip, notice, disabled send; machine rows follow | pass |
| Machines: select each computer, profile follows; Settings: each page, breadcrumb follows | pass |
| Dark theme | pass |
| Narrow: rail and pane hidden, menu button shows, header keeps the title and a short status, tokens and toggle hidden | pass |
| Drawer: opens, backdrop, Escape and choosing a page close it, choosing a section keeps it open | pass, after a fix (choosing a section closed the drawer) |
| The chat header still has the folder line, Rendered/Raw and tokens, and only claude shows a dollar figure | pass, after adding them (the first version had dropped them) |
| Console errors | none |

Not checked: real hover on touch devices, real screen readers, and any behaviour on real data (nothing here calls the API).
