# Agent and model picker — handoff

| | |
|---|---|
| **Prototype** | `model-picker.dc.html` |
| **Provenance** | The owner, 2026-09-28, with a screenshot of another app's picker: "I want the agent and model selector to look like this, which have one button to expand and select the provider and model." |
| **Mode** | build — from his reference, with what our backend can't show taken out (see Not included) |
| **Status** | reviewed 2026-09-28: the owner approved it and asked for it in the app (D-058) |
| **Design system** | the Claude artifact linked from `CLAUDE.md` |

## What this is

The two pickers under the message box (agent, then model; D-055) become one button showing the
agent's mark, its name and the model. It opens a panel with the agents down the left, the chosen
agent's models on the right, a search box across the top that looks through every agent's models,
and a starred list that holds models from any agent.

## Component mapping

| Prototype element | Use | Notes |
|---|---|---|
| The button | ghost `Button` in `composer.tsx`, where the two triggers are today | One trigger replaces two. The model name truncates first (B-51) |
| The panel | shadcn `Popover`, **not** `DropdownMenu` | It holds a text field, a tab rail and a list; a menu's roving focus fights the search box. `Popover` is in neither `components/ui` nor the design system yet: add it to both first |
| Search box | plain input, no border | Has focus when the panel opens |
| Agent rail | `ProviderMark` in icon buttons, `role="tablist"` | Star above a separator, then the agents in the daemon's order |
| Model rows | new `ModelOption` row (list option) | Label, optional description, star, check |
| "Not installed" | existing `Status tone="warning"` | Same wording as Machines |
| Switch banner, "Keep claude" | existing, in `composer.tsx` | Unchanged |

## Token usage

Existing tokens only: `--popover`, `--border`, `--muted`, `--muted-foreground`, `--foreground`,
`--ring`, `--warning`, `--warning-text`, `--provider-*`, `--primary`.

| Needed | Exists? | Action |
|---|---|---|
| A highlight that shows on `--popover` | yes, since B-59 | `--accent` used to equal `--popover` in all four dark surfaces (docs/Bugs.md B-59). Fixed in `themes.css`; the prototype uses `--accent` again |
| A shadow for a floating panel | no `shadow-md` token in `tokens.css` | Uses the same `color-mix` shadow as the composer prototype. The real `Popover` brings its own |

## States

| State | Reachable | Notes |
|---|---|---|
| Populated | default | claude open, 4 models; agy has 14, grouped by maker |
| Empty: no favourites | Favourites → None, or unstar all three | "No favourites yet" and one sentence on how to add one |
| Empty: search finds nothing | type `xyz` | Says search covers every agent; Clear search |
| Loading | State → Agents loading | The button is a skeleton, as today |
| Error: an agent can't run | State → agy not installed | agy stays in the rail with a warning badge; its page says why and what fixes it |
| Unreachable / no computer / running | State buttons | The button is disabled, as the pickers are today |
| New conversation | Conversation → New | No catch-up cost is quoted, and choosing another agent switches straight away |

## Data contract

| Field | Source | Deliverable? |
|---|---|---|
| Agents, availability, reason | `Provider.availability`, `unavailableReason` | yes — today |
| Model id and label | `Provider.models` | yes — today. claude and agy list their own; codex is our list (G-8) |
| Model description | claude's `list_models` reply, `description` | **yes for claude.** Built: `protocol.Model.description`, filled by the daemon from `list_models`. codex and agy have none, so their rows are one line |
| Which descriptions are real | — | All four, from the owner's account's `list_models` reply (`server/internal/agent/testdata/claude-list-models-families.jsonl`) |
| Limit line under the agent | `Provider.headroom` | yes for claude; codex and agy say "No limit data", as the strip does |
| Catch-up cost of switching | what the switch banner already shows | yes — the same figures, shown before choosing instead of after |
| agy's maker groups | the label's first word | derived, not reported. A label that starts with anything else lands in "Other" |
| Favourites | `favourite_models` table, `GET`/`POST /api/favourite-models` | Built: a per-account list of `provider:modelId`, kept with the account like appearance so it follows him to any browser. A starred model the computer no longer offers is hidden, not deleted |

## Interactions

- Opening focuses search; ↑ ↓ move through the rows, Enter picks, Esc clears the search and then
  closes, returning focus to the button. Clicking outside closes. Real behaviour, keep.
- Opening shows the agent the conversation is on, with the current model highlighted.
- Picking another model of the same agent changes it at once. Picking another agent in a
  conversation with messages raises today's switch banner, charged on send; in a new conversation
  it just changes. Same rule as today (B-50).
- The star toggles in place without closing the panel. Local state only — **needs the store above**.
- The + button is inert here; it is today's attach menu.

## Invented

- One trigger showing mark, agent and model (his reference shows only the model; ours names the agent
  too, because agy runs Claude models and "Claude Sonnet 4.6" alone would read as claude).
- The favourites list and its star at the top of the rail, and that it starts on the conversation's
  agent rather than on favourites.
- Search matching the model id as well as the label ("claude" finds claude's models and agy's Claude
  models).
- Grouping agy's models by maker, and the note under them that its Claude models run through agy.
- The codex note that its list is ours and the plan may not offer every model.
- Quoting the catch-up cost inside the panel, under an agent you haven't used in this conversation.
- A fixed 440px height, so moving between agents doesn't make the panel jump; 440px wide.

## Open questions

- Should the panel remember the last rail tab (e.g. open on Favourites once he uses it)? Rendered:
  always opens on the conversation's agent.
- codex's reasoning effort is a separate setting (Capabilities), unlike agy's where it is in the
  model. Not shown here; a later row in this panel is the natural home.

## Not included

From his reference, left out because nothing reports it (Capabilities.md):

- **Upgrade banner and greyed-out locked models** — no plan tiers; claude leaves out models the
  account can't use, and the daemon drops them.
- **Price tier ($$, $$$)** — no provider reports a price per model; claude reports dollars per turn
  only after it runs.
- **NEW badge** — no CLI says which models are new.
- **Capability icons (sees images, reasons, uses tools)** — known per agent, not per model.
- **Info button and filter button** — nothing to show or filter by beyond the above.

## Verification

2026-09-28, in the Browser pane, served by the `proto` launch config (`http://localhost:4181`),
dark and light, wide and narrow pane. No console messages at any point.

- Opens above the button, 440×440, search focused; the current model is checked and highlighted.
- Rail: favourites, claude, codex, agy each show their page; agy scrolls through 14 models in three
  maker groups with the note at the end; codex shows its note; the panel keeps its size throughout.
- Search "claude": claude's four (matched by id) and agy's two Claude models, match underlined;
  ↓ ×4 then Enter picked "Claude Sonnet 4.6 (Thinking)", which raised the agy switch banner
  (14 messages, ~9.8k tokens) and the button read "agy Claude Sonnet 4.6 (Thinking)".
  "Keep claude" undid it.
- Search "xyz": the no-match state; Clear search restored the list.
- Unstarring all three favourites gave the empty state.
- Esc with text clears the search and keeps the panel; Esc again closes and focuses the button.
  A click outside closes.
- Running, unreachable and no computer disable the button and it won't open; agents loading shows
  the skeleton.
- agy not installed: warning badge on its rail mark, "Not installed" with the sentence, no models
  and no catch-up cost.
- New conversation: no cost line, and picking a codex model switched straight away with no banner.

Found and fixed during the pass: the clear button showed with an empty search; the panel changed
height between agents (now fixed); the replay cost showed under an agent that isn't installed; the
highlighted row was invisible in dark (B-59, worked around here).

2026-09-28, after the owner's review: the list's scrollbar was light in dark mode. The prototypes'
`tokens.css` never declared a colour scheme (in the app, next-themes sets `color-scheme` on `<html>`,
checked on `localhost:3000`: `color-scheme: dark` inline). Added there, so every prototype gets it;
the agy list's scrollbar now draws dark, and `color-scheme` computes `dark` / `light` with the theme.
