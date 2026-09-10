# Element inventory

The menu for SKILL.md Step 2 — used when the owner has no ready answer to "what
do you picture this app containing?"

**Do not read this file at the owner.** It is roughly seventy options; reciting
it produces the same vague shrug as any long abstract question. Pick the two or
three groups the product obviously needs, offer three to five options from each,
and ask them to sort into **expect / not building / not yet**.

Everything here is a *question*, never a recommendation to build. An element
nobody asked for costs a screen, a state matrix, and a maintenance burden
forever.

## When the owner goes blank — forcing questions

These get concrete answers from people who cannot answer the abstract version.
Each one decodes into inventory without ever using the word "component".

| Ask | What it decodes to |
|---|---|
| "You open the app on Monday morning — what's the first screen, and what's on it?" | The home surface: dashboard vs list vs empty workspace |
| "You click a row in that list. What happens?" | Full page · side panel · centre modal · inline expand — the single highest-consequence answer in this file |
| "Two of those open at once — how do you get between them?" | Tabs · back-navigation · split view · nothing (one at a time) |
| "What can the user change about how it looks?" | Theming: light/dark, brand accent, surface character, density |
| "Something's running and takes 30 seconds. What's on screen?" | Progress, streaming output, toast, live status indicator |
| "It went wrong. Where do they find out, and where do they look next?" | Error surface, banner vs toast, log/history view |
| "Which screen would you show someone to explain the product?" | The hero surface that gets designed first |
| "What's in the app that you'd be annoyed to see, if I built it uninvited?" | The **not building** list, which is the one nobody volunteers |

## 1. Shell & navigation

| Element | What it is | When it earns its place | Cost / trap |
|---|---|---|---|
| Sidebar nav | Persistent left rail of top-level destinations | More than ~4 destinations, visited constantly | Eats horizontal room; collapsing it is a whole extra state |
| Top nav | Horizontal bar of destinations | Few destinations, content wants full width | Doesn't scale past ~6 items |
| Entity tab strip | One tab per open record, app-level | Users compare or juggle several records at once | Every tab must preserve its own state, and labels must be unique on screen |
| In-record side sub-nav | Sections *within* one open record | Records deep enough to have sections | Constantly conflated with the tab strip — write down what each answers |
| Breadcrumb | Path back up a hierarchy | Real nesting three levels or deeper | Decorative on a flat app |
| Command palette | Fuzzy keyboard jump to anything | Power users, keyboard-first, many destinations | Needs every action registered and named, forever |
| Global search | Search across entities | Data volume beyond scrolling | A search with no result-ranking story feels broken fast |

## 2. Theming & personalisation

| Element | What it is | When it earns its place | Cost / trap |
|---|---|---|---|
| Light / dark | Two modes | Near-default for anything used long | Every rule must hold in both — verify, don't assume |
| User-picked brand accent | The user chooses the accent hue | Product is theirs, not yours; multi-tenant | Forbids hardcoded colour anywhere; needs a contrast floor per accent |
| Surface character | Named surface treatments (e.g. paper / mono / glass) | The owner wants variety without re-deciding the palette | The plainest variant is the honest worst case — design for it, not the flattering one |
| Density toggle | Compact / comfortable / roomy | Same app serves scanners and readers | Doubles every spacing decision unless driven by one token |
| Per-workspace theming | Theme follows the workspace, not the user | Multi-tenant, users switch context | Theme becomes data, with a migration and a default |

## 3. Record surfaces — how one thing opens

The answer to "you click a row, what happens" lives here. Pick a default and
name the exceptions; leaving it open is how three screens get three answers.

| Element | What it is | When it earns its place | Cost / trap |
|---|---|---|---|
| Full detail page | Its own route | The record is the work | Loses list context; needs a real back story |
| Side panel / inspector | Slides in beside the list | Peek-and-move-on, or edit-while-comparing | Cramped for anything with a reading column |
| Centre modal | Overlay, dismissible | One field, one confirmation | A modal that scrolls a long body is the wrong surface |
| Inline expand | Row opens in place | Two or three extra facts | Wrecks row rhythm past a few fields |
| Split view | List left, record right, both live | High-throughput triage | Two responsive layouts, not one |
| Full-page editor | Distraction-free authoring | Long-form content | Needs its own save/dirty/exit model |

## 4. Data display

| Element | When it earns its place | Cost / trap |
|---|---|---|
| Table with column controls | Comparing many records across many fields | Sort, filter, resize, and empty/loading per column |
| Card grid | Visual or few-field records | Wastes space for dense data |
| Timeline / activity feed | "What happened here" matters | Needs a real ordering and grouping rule |
| Live log / output stream | Long-running processes | Scroll-follow, pause, and volume limits |
| Chart | A trend, not a number | Needs its own colour and axis rules — an unowned chart palette is instant drift |
| KPI / stat tiles | A few numbers set the frame | Decorative when the numbers don't drive action |
| Tree | Genuine hierarchy | Expensive at depth; often a filtered list in disguise |

## 5. Input & action

| Element | When it earns its place | Cost / trap |
|---|---|---|
| Standard form | Anything created or edited | Validation, error, and dirty states are the real work |
| Inline edit | Quick single-field corrections | Every field needs a save/revert affordance |
| Multi-step wizard | Genuinely sequential setup | Back-navigation and partial-save are mandatory, not extras |
| Confirm dialog | Destructive or irreversible actions | Used everywhere, it trains users to click through |
| Bulk action bar | Acting on many rows | Selection state, partial selection, and undo |
| Keyboard shortcuts | Repeated daily actions | Needs discoverability and a conflict policy |

## 6. Feedback & system state

| Element | When it earns its place | Cost / trap |
|---|---|---|
| Toast | Transient confirmation | Wrong for anything the user must act on |
| Inline banner | Persistent condition on this screen | Accumulates until nobody reads them |
| Progress / streaming indicator | Anything past ~1s | Absent, users retry and double-submit |
| Skeleton vs spinner | Loading | Pick one per surface type and write the rule down |
| Empty states | Every list, always | The first screen a new user sees, and the most-skipped |
| Error surface | Every failure path | "Something went wrong" with no next action is a dead end |
| Connection / health indicator | Realtime or agent-driven apps | Silence reads as working when it isn't |

## 7. Cross-cutting

| Element | When it earns its place | Cost / trap |
|---|---|---|
| Onboarding / first-run | Non-obvious setup | Goes stale the moment the product moves |
| Notifications centre | Events matter after they happen | Read/unread state is a data model, not a UI |
| In-app help / knowledge surface | Concepts users must learn | Becomes a lie unless it ships with the feature |
| Audit / history | Who changed what | Needs retention and permission answers |
| Settings | Any preference at all | Fills with abandoned toggles; ask what may *not* be configurable |

## Writing the result down

Three lists, into the doc's expected-surfaces section:

| List | Why it matters |
|---|---|
| **Expected** | What agents may build without asking again |
| **Not building** | Stops the same rejected idea being re-proposed every third screen |
| **Not yet** | Goes to "deliberately undecided" — an agent hitting one asks rather than inventing |

For any two elements that could be mistaken for each other, write one line each
saying what question that element answers. That line is worth more later than
either element's own description.
