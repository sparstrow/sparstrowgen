# App shell restyle after the Claude Design kit — 2026-09-19

These directions were rendered as mock-ups from the real Mono tokens and the app's real screens (Chat,
Machines, Settings), not generated as pictures: the kit supplies the exact shell, so a generated image
would have invented its own visual language, and this cost none of the owner's agent quota. They show
composition, hierarchy and density; they have no hover, no collapse and no narrow layout, which the
prototype decides.

| | Direction | Optimised for | Outcome |
|---|---|---|---|
| A | Icon rail (labels on hover) plus an always-visible pane for every section, one 56px header line across the top | The kit as drawn; one pattern every future section (agents, schedules, pipelines) can reuse | **chosen** — "Go with A" |
| B | One labelled sidebar; the active section opens inside it, no second pane | The widest content area; closest to how claude.ai does it | not chosen |
| C | Labelled sidebar, a pane only where a section has a list (Chat, Settings) | The smallest change from today | not chosen |

## What this told us about taste

Only what was actually said: the owner wants the app to look like the kit, "where the sidebar is there and
the navigations" (L-30), and picked the kit's own shape over the two variations. No reason was given for
rejecting B or C, so none is recorded; the recommendation for A (it is the kit, and a pane per section is
the slot agents and schedules will fill) was the agent's.
