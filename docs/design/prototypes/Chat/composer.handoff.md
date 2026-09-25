# Composer — handoff

| | |
|---|---|
| **Prototype** | `composer.dc.html` |
| **Provenance** | The owner, 2026-09-24: "if I type a block of text, there is an empty space which can be expanded for the text. The agent and model selector is in the way." |
| **Mode** | explore — three directions, and the composer as it is today for comparison |
| **Status** | draft, waiting for the owner to pick a direction |
| **Design system** | the Claude artifact linked from `CLAUDE.md` |

## What this is

The message box at the bottom of a conversation. Today the agent and model pickers sit to the left
of the text, so a long message wraps in the right-hand two thirds and leaves the space under the
pickers empty. Each direction gives the text the full width.

- **A · Controls under the text.** Text on top, full width. One row underneath: agent, model, then
  send on the far right. The row is there when the box is empty too, so nothing moves when typing
  starts.
- **B · Controls above the text.** The box opens with a "To" line naming the agent and model. The text
  runs full width under it. Only the send button keeps a column, at the bottom right.
- **C · Pick the agent in the strip.** The agent strip above the transcript becomes the picker, with
  the model beside the chosen agent. The box holds only the text and the send button.

## Component mapping

| Prototype element | Use | Notes |
|---|---|---|
| Agent and model pickers | existing `DropdownMenu` + ghost `Button`, as in `composer.tsx` | Unchanged behaviour. They only move |
| Text box | the existing `textarea` | Its height cap changes, see Invented |
| Send / stop | existing `Button size="icon"` | Unchanged |
| Switch notice, unavailable notice | existing, in `composer.tsx` | Unchanged |
| C: pickable agent chips | `provider-strip.tsx`, made into buttons | **New behaviour** for an existing surface |

## Token usage

Only existing tokens: `--background`, `--card`, `--muted`, `--accent`, `--border`, `--ring`,
`--primary`, `--secondary`, `--popover`, `--warning*`, `--provider-*`. Nothing new is needed.

## States

| State | Reachable | Notes |
|---|---|---|
| Empty box | `?text=empty` | A keeps its control row; B keeps its To line |
| One line | `?text=short` | |
| Long message | `?text=long` | The owner's case: a pasted vendor note |
| Running | `?state=running`, or press send | Send becomes stop, as today |
| Switch pending | `?state=switch` | The catch-up notice, unchanged |
| Unreachable | `?state=unreachable` | Amber notice, as today |
| No computer | `?state=none` | Neutral notice, as today (B-46) |
| Agents loading | `?state=loading` | Pickers wait as placeholders; the text box stays usable |
| Narrow pane | `?w=narrow` | B-51: the text must never be squeezed by the pickers |

## Data contract

Nothing new. Every direction shows the same agent, model, notices and running state as today,
from the same sources. No backend work.

## Interactions

- Agent and model menus open, choose and close. Real behaviour, keep.
- Enter sends and Shift+Enter adds a line. Sending adds the message and starts "running"; stop ends
  it. The reply itself is faked.
- C: clicking an agent in the strip switches to it; the model menu opens from the chip.

## Invented

- **The box grows to 40% of the window before it scrolls**, instead of today's 200 px (about eight
  lines, the cap the screenshot hit). All three directions use it. Today's view keeps 200 px.
- A: the control row shows when the box is empty.
- B: the word "To" and the line under it.
- C: a chosen agent's chip shows its model in place of its limit text, so the limit is hidden for
  the agent in use.
- The pasted vendor note used as the long message.

## Open questions

- OQ: should the box also get a button to open a large editor for very long messages? None of the
  directions has one; the 40% growth may be enough.

## Not included

- Attachments and slash commands: nothing in the app supports them yet.

## Verification

2026-09-24, served locally, in the Browser pane:

- A, B, C and Today each render with the long message, dark theme, wide pane.
- C with the switch pending in a narrow pane. A with the box empty, the computer unreachable, in a
  narrow pane with the light theme. B with agents loading in a narrow pane.
- A: the model menu opens upward and choosing Sonnet 5 relabels the picker. Send adds the message and
  switches to running with a stop button; stop returns to ready.
- C: clicking agy in the strip selects it with its first model.
- No console errors on any of them.
