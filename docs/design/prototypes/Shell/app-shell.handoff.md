# App shell — handoff

| | |
|---|---|
| **Prototype** | `app-shell.dc.html` (open it from disk; it links `../tokens.css`, `../icons.js` and `../seed-data.js`) |
| **Provenance** | The owner's request, "start the shell restyle like the Claude Design app" ([`Later.md`](../../../Later.md) L-30), his choice of direction A on 2026-09-19 (D-042), and his two decisions on the prototype the same day: the rail can be pinned open (D-043) and the phone gets a bottom tray with the pane as its first screen (D-044). No spec: this restyles what exists. |
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
| Icon rail, hover-expands as an overlay, pinnable | **NEW `Rail`** in a new `components/shell/` | The shadcn `Sidebar` is not reused: its icon mode pushes the content instead of overlaying it, and its `--sidebar-*` tokens are never defined in this app, so its hover and active fills paint nothing (README, "Live gaps"). Pinned is the one case where the rail *does* push the content, so it is a layout width, not an overlay. |
| Bottom tray (under 768px) | **NEW `SectionTray`** in `components/shell/` | The same three entries as the rail. Nothing in the design system covers it. |
| Pane header, list, search | **NEW `Pane`** wrapping existing content | Chat: the existing `ConversationList` (drop its own title row and "New" button in favour of the pane header). Settings: the nav from `settings-shell.tsx`. Machines: rows built from the existing machine list. |
| 56px header | **NEW `AppHeader`** | Title, optional second line, right-hand slot. |
| Computer status in the header | existing `Status` (`success`, `progress`, `neutral`) | The word is always shown; "Online" alone at narrow widths. |
| Provider strip, composer, transcript, provider chips | existing `ProviderStrip`, `Composer`, `MessageList` | Unchanged; only their surroundings move. |
| Machine rows and profile | existing `Status`, `Button` | Profile content is unchanged. |
| Settings rows and cards | existing `SettingsRow`, `SettingsCard` | |
| Buttons, chips, switch | existing `Button`, `Switch` | The Mode, Surface and Accent chips are hand-built in the app today and stay so (the design system lists a segmented control as missing). |
| Phone list and detail | the same `Pane` and section content, at full width | No sheet, no drawer: the pane *is* the list screen and the detail is the route below it. `Sheet` is no longer needed. |

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
| Narrow (under 768px) | yes (`?width=narrow`, or a narrow window) | Sections move to a bottom tray; the pane is the first screen and the detail sits below it. Every state above is reachable there too. |
| Rail pinned open | yes (the pin in the rail header, or `?pin=1`) | Desktop only. |
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
| Rail pinned open | a Zustand view-state store, persisted to `localStorage` per browser (D-043) | yes, and **no backend at all** — not a column, not an endpoint |
| Which phone screen is showing (list or detail) | the route: a list route per section and a detail route under it | yes — the URL already distinguishes `/`, `/machines/[id]` and `/settings/...`, so "list" is the section route with nothing selected |

Nothing here streams. What must survive a refresh: the selected conversation, computer and settings page are in the
URL today (`/`, `/machines/[id]`, `/settings/...`); the prototype keeps them in memory.

## Interactions

Real behaviour, keep: the rail's section switch and hover or focus expansion (overlay, no layout shift); the pin,
which holds the rail open and does push the content; the pane's selection; conversation search across title and
folder including the archive; the Archived toggle; New conversation adds an untitled row and selects it; the states;
and on a phone, the tray's section switch (which returns you to that section's list), opening a row into the detail,
and the back arrow or Escape returning to the list.

Faked, do not port: the send button, Add computer, Disconnect, the folder button, the Rendered/Raw toggle, the
Mode/Surface/Accent chips, Change password and Check now all show a note instead of acting; the transcripts and
counts are seed data.

## Invented

Everything here was decided by the prototype and approved by nobody.

- **The collapsed rail's top mark is a lowercase "s"** in plain type. The design system says the wordmark is plain
  type and there is no logo, so the kit's robot icon was not used; the full wordmark shows when the rail expands.
- **No shadow** on the expanded rail (the kit uses one).
- **The pin's icon and its place** — `panel-left`, in the rail header, revealed with the labels on hover. There is
  no pin icon in the app's icon set, and `panel-left` is what the app already uses for a sidebar toggle.
- **The tray hides itself in a conversation** and only there, so it does not sit under the composer. Every other
  detail screen keeps it. The kit shows no phone layout at all, so all of this is mine.
- **The phone's back arrow** is `chevron-left`, which had to be added to the prototype's icon set.
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
- **OQ3 — settled by D-044.** `/machines` opens the first computer on a desktop, where the list is beside it anyway, and is
  the bare list on a phone, where the list is a screen of its own. Same for `/settings`.
- **OQ4 — settled by D-043.** The pin answers it: hover-only by default, always-on once pinned.
- **OQ5:** the Chat pane's "New conversation" button is a plus in the header. The kit's convention is a header action; the
  app today has a labelled button in the conversation list.

---

# First-run setup — added 2026-09-20

| | |
|---|---|
| **Provenance** | Approved spec US2, [`2026-09-12-first-usable-release.md`](../../../specs/2026-09-12-first-usable-release.md) — the unbuilt half. Directions from [`shots/2026-09-20-first-run-setup/`](../../shots/2026-09-20-first-run-setup/README.md); the owner chose **A and B together** and said why (D-046). |
| **Mode** | build |
| **Status** | draft — waiting for the owner to react |
| **Reach it** | `?setup=idle` (and `waiting`, `unanswered`, `approval`, `ready`, `loading`, `error`); `?resume=1` for the in-app card; `?future=1` to see where later steps would sit |

Built into this same prototype rather than a new file, at the owner's request
("if I ask more to build you can start adding on same app-shell.dc, that will save lot of time and
token"). The wizard replaces the whole frame; the resume card lives inside it.

## Component mapping

| Prototype element | Use | Notes |
|---|---|---|
| `.setup` full-screen wizard | **NEW `SetupWizard`** on a route of its own (`/setup`) | No rail, no pane, no tray: there is nowhere else to be during first-run setup. |
| `.steps` stepper | **NEW `Stepper`** | Nothing in the design system covers it. Built from the step list so adding a step is a data change. |
| `.setupcard` in the Chat pane | **NEW `SetupCard`** in `components/shell/` | The compact rendering of the same list. It reopens the wizard; it never carries its own copy of a step. |
| The step list itself | **NEW `useSetupSteps()`** in `lib/queries.ts` | The single source both read (D-046). Derived from `useMachines()` today; server-backed when a step becomes an account fact. |
| Connect card, in all six states | existing `Status` (`progress`, `warning`, `success`, `danger`), `Button` | Same vocabulary as everywhere else (D-040). |
| Install help | existing `/install` page | Reached from "I have not installed it yet" and from the unanswered state. |

## States

| State | What it is | How it is reached |
|---|---|---|
| `idle` | Before anything is launched. Explains what connecting does. | Arriving at setup, or "Not now" from approval |
| `waiting` | The link was opened; polling for a claim | "Connect this computer" |
| `unanswered` | Eight seconds with no claim. Retry **and** install help, never "it is absent" | the timer (2.6s here, so the branch can be watched) |
| `approval` | The computer answered. Names it, its version and the agents found | the poll seeing `claimed` |
| `ready` | Connected. The only state with no Skip row | "Approve computer" |
| `loading` | Checking the account before offering anything | first paint |
| `error` | The pairing request could not be created at all | a failed `POST /api/machines/pairings` |

## Invented — not in the spec, not approved

- **The wizard's copy.** Every sentence on it.
- **"Step n of m" in the top bar** as well as the stepper. Two renderings of the same fact; drop one
  if it reads as noise.
- **The `?future=1` steps** ("Workspace", "Profile"), shown dashed and labelled *(later)*. They exist
  only to show the owner that the shape takes more steps. **They are out of scope for this release**
  (first-usable-release, Out of scope: "Multiple workspaces, workspace invitations and workspace
  management") and must not be built.
- **"See this computer"** on the ready step, going to the Machines profile.
- **The dismiss (×) on the resume card.** Nothing says a person may hide it; if they can, whether it
  comes back on the next sign-in is undecided.
- **The calm (non-amber) notice** for "no computer yet". Not having connected one is a step not
  taken, not a fault, so it does not get the colour that means "attention needed".

## Open questions

- **OQ6 — does the wizard run for an invited person who already has a computer?** It is reached when
  the account has none. Somebody who paired on another browser has one, so they would never see it.
  Believed right; not confirmed.
- **OQ7 — does Skip ever expire?** The resume card currently stays until setup is finished or the ×
  is pressed. Nothing says whether a dismissed card should return.
- **OQ8 — what does the wizard do on a second computer?** Today it is first-run only; "Add computer"
  in Machines stays the way to add another. Worth confirming that is what he wants.

## Deliberately absent

- **A refused pairing** — the computer belongs to another account. The browser cannot currently tell
  that apart from one that never answered ([`KnownGaps.md`](../../../KnownGaps.md) G-40), so drawing
  it would be designing a state the backend cannot serve. It needs the refusal recorded against the
  pairing first.
- **Install detection.** There is none and cannot be one: the daemon dials out only.

## Not included

- The kit's theme toggle in the header: appearance is per account and changes in Settings.
- Any destination the kit has and this app does not (dashboard, agents, projects, task board, messages, runs, pipelines,
  schedule, memory, terminals).
- The account menu's contents (the existing `AccountMenu` is reused).
- Keyboard shortcuts.

## Verification

Run 2026-09-19 with the frontend-verify loop, in the Browser pane against a static server, at 1280 wide and 390 wide.
Re-run the same day after the pin and the bottom tray were added.

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
| Narrow: rail hidden, tray shows the three sections, the pane fills the screen and carries the section name, live status and action | pass |
| Narrow: opening a row shows the detail with a back arrow; back and Escape return to the list; switching section returns to that section's list | pass |
| Narrow: the tray is hidden in a conversation and present in a computer's profile and a settings page | pass |
| Narrow: every state (populated, empty, loading, error), offline and reconnecting, and dark, reachable and correct on the list screen | pass |
| Narrow: no horizontal overflow at 390px, including the longest seeded title and the "No computer connected" header | pass |
| Pin: holds the rail at 224px with labels, moves the pane across, unpins back to 60px with the overlay behaviour intact, survives a reload via the URL | pass |
| The chat header still has the folder line, Rendered/Raw and tokens, and only claude shows a dollar figure | pass, after adding them (the first version had dropped them) |
| Console errors | none |

Not checked: real hover on touch devices, real screen readers, and any behaviour on real data (nothing here calls the API).

### First-run setup — verified 2026-09-20

Run against `http://localhost:4181/Shell/app-shell.dc.html` (the `proto` launch config, added in
this change so the next session does not have to rebuild it).

| Checked | Result |
|---|---|
| All seven setup states render their own heading, body, tone and buttons | pass — read from the DOM, not eyeballed |
| Each state in light **and** dark, at 390 wide | pass — 14/14, no horizontal overflow in any |
| `waiting` → `unanswered` on the timer | pass |
| `unanswered` → "Try again" → `waiting` → `unanswered` again | pass |
| Skip leaves the app with the resume card, in Chat | pass |
| The resume card reopens the wizard rather than duplicating its steps | pass |
| Approve → ready → "Start your first conversation" clears the card and the notice | pass |
| `?future=1` numbers correctly (Connect becomes "Step 4 of 5") | pass |
| Console | clean, no messages |

**Two bugs this found and fixed, both in the existing shell rather than the new work:**

- The composer said "Your machine is unreachable" after a skip, when the account has no computer at
  all — the same wrong sentence as [`Bugs.md`](../../../Bugs.md) B-46, which was found independently
  in `apps/web` the same day. Both now say nothing is connected yet.
- The provider strip rendered its empty container in that state. It is hidden, matching B-45.

**Not verified:** nothing here has been seen by the owner, and none of it exists in `apps/web` yet.

### First-run setup — wired into apps/web, 2026-09-20

The owner approved the Machines step and set the order (D-047), so it was built. Against the real
backend on a local stack: a seeded account with no computers, a real pairing, and a real daemon
claiming it.

**Where it lives**

| Piece | File |
|---|---|
| The wizard | [`components/setup/setup-wizard.tsx`](../../../../apps/web/components/setup/setup-wizard.tsx) |
| The resume card | [`components/setup/setup-card.tsx`](../../../../apps/web/components/setup/setup-card.tsx) |
| The step list both read | `useSetup()` in [`lib/queries.ts`](../../../../apps/web/lib/queries.ts) |
| The pairing sequence both surfaces run | [`lib/pairing.ts`](../../../../apps/web/lib/pairing.ts) |
| Skipping, per browser | `useSetupView` in [`lib/store.ts`](../../../../apps/web/lib/store.ts) |
| The route | [`app/setup/page.tsx`](../../../../apps/web/app/setup/page.tsx) |

**Verified against a real server and a real daemon**

| Checked | Result |
|---|---|
| Signing in with no computer lands on `/setup` | pass |
| "Connect" mints a real pairing and polls it | pass — `POST /api/machines/pairings`, then `GET` every 1.5s |
| Eight seconds with no answer offers retry **and** install help | pass |
| A daemon claiming the request **after** that offers approval anyway | pass — see the bug below |
| The approval prompt names the computer | pass — "Approve DESKTOP-GJ8NLB8?" |
| Approving connects it and shows the final screen | pass |
| The final screen fills in agents and version live when the daemon connects | pass — "claude, codex, agy", version `dev`, no reload |
| Skip leaves the resume card in the Chat pane and sets the browser flag | pass |
| The card reopens the wizard and clears the flag | pass |
| Finishing clears the card and `/` stops redirecting | pass |
| Reaching `/setup` with nothing outstanding | pass — says so; does not offer to connect again |
| 375x812 | pass, no overflow |
| Console on a clean load | clean |

**Two bugs found by running it, both mine, both fixed before merge:**

1. **Polling stopped at eight seconds.** Extracting the sequence into `usePairing` tied the poll to
   the `waiting` stage, so it was torn down at the exact moment the screen says the computer "may
   still be starting". A daemon that claimed the request at nine seconds was never noticed and the
   person sat on "has not answered" forever. The poll now runs through `unanswered` too; only the
   countdown belongs to `waiting`, in its own effect so "Try again" restarts it.
2. **The final screen could not hear anything.** The wizard replaces the shell, and the shell is
   what mounts the one websocket — so `/setup` had no socket at all and "Agents found: none
   reported yet" would never have changed. `SetupWizard` now mounts it; still exactly one caller,
   because that route never mounts `AppShell`.

**One thing the design asked for that the backend did not serve.** The prototype named the computer
in the approval prompt; `GET /api/machines` returns approved computers only, by design, so the
browser had no name. Rather than drop it — an approval prompt that names nothing is a weak thing to
ask someone to agree to — the pairing now carries `machineName`, which is where the query's own
comment says a pending computer belongs.

**Still not done:** OQ6, OQ7 and OQ8 above are unanswered, and none of this has been seen by the
owner or run on production.

---

## Appendix — Workspaces, and setup step two (2026-09-21)

The owner answered what a workspace is: *"Workspace is when I want group of projects, skills,
chats, separate. I would create one personal and one work related workspace. Keeping both of them
Separate. ALso adding other people to my workspace in future. Check how multica did the workspace,
and implement it."* That closed the last open step in his order (`docs/Decisions.md` D-047), and
the ownership trade it forced is D-050.

**What this prototype gained**

| Piece | Where |
|---|---|
| The switcher at the top of the rail | `switcher()` — above the sections, because it scopes them |
| Settings → Workspaces | `mainset()`, id `workspaces` |
| Settings → Account | `mainset()`, id `account` — the profile that shipped in #58 and was never drawn here |
| Step one and step two of the wizard | `profileStep()`, `workspaceStep()` |
| Three real steps in the stepper and the resume card | `setupSteps()`, `setupCard()` |

The toolbar's **Later steps** toggle is gone. It existed because Profile and Workspace were drawn
as things that did not exist yet; both are built, so the toolbar now has a **Profile** and a
**Workspace** toggle instead, each switching that step between done and outstanding.

**Decisions this made, all of them now in D-050**

- The switcher is **above** the sections, not one of them. Chat is a place you go; a workspace is
  the world you are in when you get there.
- It is shown with **one** workspace as well as several. Hiding it until there are two means the
  first switch happens through a control nobody has ever seen, and it is also the only place that
  says which workspace the conversations on screen belong to.
- **Making and renaming are not in the menu.** A switcher that also creates is a menu you cannot
  open without risking the thing you did not mean to do. Both live in Settings → Workspaces, which
  the menu's last entry goes to.
- **Step two cannot be skipped.** Every other step can wait; a workspace cannot, because every
  conversation is kept in one and skipping would land somebody on a Chat with nowhere to put
  anything. The skip row says that instead of offering a button.
- **"Do this later" still is not "done".** Passing over step one leaves it outstanding on the
  resume card. The prototype models that with its own `passed` map, because getting it wrong here
  would have made the stepper claim something the account had not done.

**What the real app does that this cannot show:** switching workspaces re-keys the conversation
query, so the list, the search and the recent folders all change with it. The prototype's switcher
only toasts — there is one seeded conversation set, and giving it two would say more about the seed
data than about the design.

### Verification — the real app, against a real server

| Checked | Result |
|---|---|
| An existing account's conversations land in a backfilled "Personal" | pass — migration 00017, 1 conversation moved |
| The switcher lists the account's workspaces and marks the open one | pass |
| Switching to an empty workspace empties the conversation list | pass — "No conversations yet" |
| Switching back restores the first workspace's conversations | pass |
| Settings → Workspaces renames in place and opens another | pass |
| A brand-new account with **no** workspace is sent to `/setup` even though this browser had already skipped setup | pass |
| Step two has no "Skip for now"; the row explains why | pass |
| Creating the first workspace advances to step three and the skip returns | pass |
| Light and dark, and 375x812 | pass, no overflow |
| Console on a clean load | clean |

**One bug found by running it, mine, fixed before merge:** the "Add a workspace" box opened
pre-filled with "Personal" — the suggestion meant for the *first* one — so typing a second name
appended to it and offered to create "PersonalWork". The suggestion now belongs to the empty-account
case only.

**Not verified:** pressing Enter in either name box. The browser tool's synthetic Enter does not
trigger a form's implicit submission — confirmed by the same failure on the shipped registration
form, which is a control this change did not touch. See `docs/Unverified.md` U-25.
