# Raw exchange — handoff

| | |
|---|---|
| **Prototype** | [`raw-exchange.dc.html`](raw-exchange.dc.html) — `?state=populated\|running\|not-recorded\|never-ran\|truncated\|loading\|error`, `&theme=dark\|light` |
| **Spec** | [`docs/specs/2026-09-23-raw-exchange.md`](../../../specs/2026-09-23-raw-exchange.md) (Approved 2026-09-23) |
| **Provenance** | `build`. The owner approved the spec and asked for the design and the build in one go while he was away ("you can design proto and also build this into app. I'll review this working feature later"). So this direction was **chosen by the agent, not picked by him from options**. See Invented. |
| **Status** | built into the app on 2026-09-23; awaiting the owner's review of the working feature |

## What this is

The Raw view, extended from "the saved text of each message" to the whole exchange behind each agent
turn: what was sent to the CLI, what the CLI said it loaded, and every line it printed, in order and
unedited. It is for checking, not reading. It answers "did we render that properly?", "what was the
agent actually given?" and "what did it do?".

## Shape

Each agent turn in Raw keeps today's header and saved text. Between them sits one quiet line,
**sent and received · 61 lines · 22.0 KB · 8.2s**, in the same place the activity line sits in the
rendered view ([`agent-activity`](agent-activity.handoff.md)). Opening it shows three sections:

1. **Sent to claude**: the command line exactly as launched, the folder, whether a session was
   continued, and what went in on stdin. The stdin has two views: *As the agent reads it*, the prompt
   text including any catch-up, and *Exact bytes*, the stream-json envelope as written.
2. **What claude reported loading**: whatever the CLI said about itself in its first line (version,
   model, permission mode, tools, skills, agents, MCP servers, folder, session), and its usage: read,
   of which from cache and written to cache, wrote, of which reasoning, and the total. One sentence
   explains why "read" dwarfs what you sent.
3. **Received**: one row per line, `time · kind · the line`. A run of streaming fragments of one kind
   collapses into one row with a count and the text they spell (`content_block_delta · text_delta ×18
   "```json { "name"…"`), and opening it lists every line in it. Opening any line shows it formatted,
   or exactly as printed. stderr lines are marked `stderr`. **Copy lines** copies the JSONL as
   received.

When the exchange is open, the saved text below gets a label, "saved answer · what Rendered shows",
so the two are not confused.

Only the latest turn starts open.

## Component mapping

| Piece | From |
|---|---|
| Transcript lines, provider colours, saved text | today's `RawTranscript` |
| The disclosure line and its trail | the agent-activity pattern (chevron, 1px `border` trail) |
| Formatted / Exact and the stdin switch | the Rendered / Raw two-button pattern (`ViewToggle`) |
| "Not recorded", "never started", "still working", truncation | `Status` (`neutral`, `progress`, `warning`) or a quiet line with `circle-minus` |
| Load failure | `Status` `danger` + a Try again `Button` |
| Loading | `Skeleton` rows in the real row geometry |
| Tool / skill lists | `Badge` `outline` |

## Token usage

`muted-foreground` for times, previews and keys; `foreground` for labels and values; `accent` on
hover, `muted` for an open row; `card` + `border` for a line's own box. Provider colours for names
only. No new tokens.

## States

| State | What it shows |
|---|---|
| populated | an agy turn (closed) and a claude turn after a switch (open): the catch-up in stdin, 61 real lines |
| running | the claude turn's real lines arriving; "so far" in the disclosure, "still working" under the rows, "Nothing yet" for loaded until the first line |
| not recorded | a turn from before recording: one quiet line saying so, then the saved answer |
| never ran | the command that would have run, "codex never started", nothing received, the failure |
| truncated | the rows, then "Recording stopped at 16 MB for this turn" with how much was not kept |
| loading | skeletons inside the opened exchange |
| error | "Could not load this exchange", the saved answer still readable, Try again |

## Data contract

Per agent turn, stored with the conversation (and deleted with it), streamed while the turn runs.
`GET /api/turns/{entryId}/exchange`:

| Field | Type | Source | Deliverable |
|---|---|---|---|
| `recorded` | bool | whether the daemon sent anything for this turn | yes |
| `sent.program`, `sent.args` | string, string[] | the daemon, as launched | yes: we build them |
| `sent.cwd`, `sent.resumeSessionId` | string | the daemon | yes |
| `sent.prompt` | string | the daemon's built prompt, catch-up included | yes |
| `sent.stdin` | string | the exact bytes written to stdin | yes |
| `sent.launched` | bool | false when the daemon refused before starting the CLI | yes |
| `lines[]` | `{seq, atMs, stream: stdout\|stderr\|log, text}` | every line the CLI printed, stamped with ms since launch | yes |
| `dropped` | `{lines, bytes}` | lines not kept past the per-turn cap | yes |
| `report.*` | version, model, permissionMode, cwd, sessionId, tools, skills, agents, mcpServers | **computed on read** from the lines, per provider | claude: all. codex: session only. agy: session and folder |
| `report.usage` | `{input, fromCache?, toCache?, output, reasoning?, total}` | computed on read from the last usage the CLI reported | yes, all three (Capabilities) |

`report` is derived, never stored, so a better reading of old lines improves old turns too.

Live: the browser gets `exchange` events carrying the sent record, new lines (by `seq`) and the current
report. A gap in `seq` means refetch.

## Interactions

- Open and close a turn's exchange, a group, a line: real behaviour, keep.
- Formatted / Exact, and stdin As read / Exact bytes: keep.
- Copy lines: copies the JSONL as received; keep.
- The running replay and the state bar are prototype controls.

## Invented

- The whole direction. There was no shots round: the owner delegated it.
- Placing the exchange between the header and the saved answer, closed except for the latest turn.
- Collapsing runs of streaming fragments into one row.
- Naming claude's `stream_event` rows by the event inside them.
- The 16 MB per-turn cap and its wording.
- The usage breakdown's words (read, from cache, written to cache, wrote, reasoning).

## Open questions

- Whether he wants every turn open, or a way to open all. Rendered as: only the latest.

## Not included

- The agents' built-in instructions word for word: no CLI reports them.
- Editing and re-sending what was sent: out of scope in the spec.
- The tidied step-by-step view: [`agent-activity`](agent-activity.handoff.md).

## Verification

2026-09-23, in the browser pane at 658px wide, dark and light:

- populated: 61 claude lines render; a text_delta group opens to its six lines; stdin switches views.
- running: lines arrive and the disclosure reads "so far".
- not recorded, never ran, error: each renders its copy; never ran first said "0 lines · 0 B · 0.0s",
  now "sent · codex never started".
- Row labels first read `stream_event · content_block…`, cutting off the part that matters; they now
  name the inner event.
- Console: no errors.
