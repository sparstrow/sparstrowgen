# Agent activity — handoff

| | |
|---|---|
| **Prototype** | [`agent-activity.dc.html`](agent-activity.dc.html) — `?state=populated\|running\|empty\|loading\|error`, `&theme=dark\|light` |
| **Spec** | [`docs/specs/2026-09-24-agent-activity.md`](../../../specs/2026-09-24-agent-activity.md) (Draft) |
| **Provenance** | `build` — from the owner's reference component (`ThinkingState`, four variants) and his screenshot of "Worked for 37s ⌄" above a final answer |
| **Status** | awaiting the owner's reaction |

## What this is

What an agent did in a turn, inside the transcript. While the turn runs, the design system's
`WorkingIndicator` is the header, and the steps grow beneath it. When the turn ends, the steps fold
into one line, "Worked for 37s ⌄", above the final answer. Opening that line shows every step, and
a step with more to show opens in place.

## Component mapping

| Piece | From |
|---|---|
| Header while running | `WorkingIndicator`, unchanged: pixel grid, shimmer, timer |
| Turn header, answer, usage | `AgentTurn` |
| A refused step | `Status` `warning` ("Not allowed" plus the reason): a person must act, nothing broke |
| A failed turn | the existing failure notice, with the trail kept above it |
| Icons | Lucide: `brain`, `file-text`, `search`, `file-pen`, `square-terminal`, `chevron-down`, `loader-circle` |
| The folded line and the step rows | **new**: not in the design system yet, see Open questions |

## Token usage

`muted-foreground` for verbs and the folded line; `foreground` for the object of each step (the
path, command or query, which is what you scan for); `accent` on hover; `muted` for an opened step;
`card` and `border` for a step's detail. `success-text` / `destructive-text` for +/− line counts and
`exit 1`, always with the sign or the words, never colour alone. No new tokens.

## States

| State | What it shows |
|---|---|
| populated | two finished turns, folded; claude's has 9 steps including a failing command and an edit, codex's has a web search and a refusal |
| running | a third turn whose steps arrive every ~1.4s under the working indicator, the current one spinning; then it settles into exactly the finished form |
| empty | a turn that only answered: **no activity line at all**, not "Worked for 0s" |
| loading | the conversation's skeletons; the trail loads with the transcript, not separately |
| error | the turn broke after 5 steps: folded line, partial answer, failure notice; the steps still open |

## Data contract

Per agent turn, **stored with the transcript** (scenario 6: it has to survive a refresh and show on
another device), streamed while the turn runs.

`activity: Step[]`, in order, each:

| Field | Type | Notes |
|---|---|---|
| `kind` | `thought` \| `note` \| `read` \| `search_code` \| `edit` \| `run` \| `search_web` \| `refused` \| `other` | `other` keeps an unknown tool visible under its own name (AGENTS.md §3) |
| `state` | `running` \| `done` \| `failed` | a command that exited non-zero is `failed`, not a refusal |
| `startedAt`, `endedAt` | timestamps | "Thought for 4s"; the folded line's total is the turn's own duration |
| `path` | string? | read, edit |
| `command` | string? | run |
| `query` / `pattern` | string? | search_web / search_code |
| `added`, `removed` | int? | edit, counted from claude's `old_string` / `new_string` |
| `detail` | string? | the diff for an edit, the output for a command, **bounded**: a command printing 40 MB must not become a 40 MB row |
| `text` | string? | a `note`: the agent's narration between steps |
| `reason` | string? | a refusal, in the agent's words |

Per provider, from docs/Capabilities.md "Agent activity":
- **claude** — from `tool_use` / `tool_result`, the thinking block, and `text` between tools.
- **codex** — from `item.started` / `item.completed` by item type. Its refusals appear on stderr only.
- **agy** — from `step_update` (`tool_name`, `tool_info.parameters`, `state`, `duration_seconds`)
  and `result.denied_actions`.

**The final answer is the last `text` / `agent_message` / `result.response`.** Earlier text becomes
`note` steps. That is what makes "only the final response" true without losing anything.

## Interactions

- The folded line toggles the trail; the chevron turns. Folded by default for every finished turn,
  including one that just finished while you watched.
- Read, search-code, edit and run steps open their detail in place; thought, web search and refused
  steps have nothing more to show and are not buttons.
- The running trail is open and cannot be folded while it runs.

## Invented

Decisions nobody has approved yet:

1. **Folded even when you watched it happen.** Your screenshot shows a finished turn folded; I
   assumed that holds for the turn you just watched too.
2. **Intermediate prose goes into the trail** as muted `note` lines, and only the last message is
   "the answer".
3. **A read shows no file contents** in its detail — only the path and line count — so that
   transcripts do not become copies of the repository.
4. **Web searches show the query only**, because no agent reports sources. Your reference's
   "sources read" and "+7 more" were dropped for that reason, not for taste.
5. **No thinking text.** "Thought for 4s" only, because none of the three agents hands the text
   over. Your reference's Reasoning variant cannot be served.
6. **No per-step durations**, only the total. Less noise; the data exists if wanted.
7. **Refusal is amber (`warning`)**, not red: nothing broke, a permission is missing (L-33).

## Open questions

- L-33: what the agents may do. The design shows refusals honestly either way.
- The folded line and step row are new components. Once approved, they go into the design system as
  `ActivityTrail` (U-24 is the same kind of drift).

## Not included

Stopping a single step; re-running a command; opening a file from a step; any cost per step.

## Verification

2026-09-24, in the Browser pane, against this file served locally:

- **populated:** both turns folded ("Worked for 37s", "Worked for 21s"). claude's opened to 9 steps.
  The edit's detail showed its 2 removed and 4 added lines, and the failing run showed `exit 1`.
  codex's showed the web search and the amber "Not allowed" refusal.
- **running:** steps arrived under the working indicator with the current one spinning. After the
  last, the turn settled to "Worked for 8s" plus its answer, and that folded line opened to 5 steps.
- **empty:** no activity line rendered.
- **loading:** 5 skeletons in the transcript's geometry.
- **error:** folded line, partial answer, failure notice; opened to 5 steps.
- **Theme:** light and dark. At 375px, paths truncate, with no horizontal overflow.
- **Console:** no errors in any state.

Not checked: reduced motion, which the pane cannot emulate. The rules are present as they are in
`globals.css`.
