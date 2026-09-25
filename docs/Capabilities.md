# Capabilities — what the backend can actually deliver

**Read this before designing anything.** Design-driven development only works if the design stays
inside what the backend can produce. A screen showing data no provider emits is waste — it looks
finished, the owner approves it, and then the backend can't serve it.

This file is the feasibility surface. It is not a plan and not a roadmap: it says what is
*possible*, not what is scheduled.

Mark every claim as **verified** (we ran it and saw real output) or **assumed** (looks true from
documentation or flags, not yet proven). Downgrade nothing silently — if something assumed turns
out false, fix it here in the same change that discovers it.

**A verified row has a shelf life.** These CLIs ship constantly — `codex` went 0.153.4 → 0.154.0
and `agy` 1.1.27 → 1.2.0 within a day of first being captured. Keep the version numbers in the
table current, and when something behaves unlike its row, suspect the version before suspecting
the row.

---

## Shape of the system

The agent CLIs run on the owner's machine, next to the code. The server runs on a VPS and holds
the database. The browser talks to the server, never directly to the machine.

Consequences a designer must respect:

- **Anything touching the local filesystem, a local process, or localhost goes through the daemon.**
  If the daemon is offline, that part of the UI has no data. Every surface that depends on the
  machine needs an offline state — this is not an edge case, laptops close.
- **Round-trip latency is browser → server → daemon → CLI.** Fine for streaming text. Not fine for
  anything designed to feel instantaneous on keystroke.
- **The browser can reach the machine's localhost dev server** through a reverse tunnel over the
  daemon's existing connection. *(assumed — designed, not built)*
- **A provider CLI can be installed but unusable** — no signed-in account, expired auth, an
  unexecutable binary. This is not the same as offline: offline resolves itself when the machine
  comes back, unusable does not resolve until a human acts. See "Provider availability" below —
  the distinction is adopted from Multica and it is not optional to collapse the two.

## First usable release: account, pairing and updates

These rows separate what exists today from what the approved first release still has to build.
The design may show the intended journey, but it must not present an unbuilt state as live proof.

| Capability | Evidence |
|---|---|
| Email-and-password sign-in with revocable browser sessions | **Verified in this repository.** Sign in, sign out, sign out everywhere, change password, and sessions ended by a password reset. |
| Invitation-only multi-user registration without a server-log setup code | **Built and tested 2026-09-12.** `OWNER_EMAIL` plus `ALLOWED_EMAILS` may register; the setup code and the one-account index are gone (D-032). API tests and a browser walk against the real server. Existing accounts and conversations survive the migration, verified on a copy with one account and refused on one with none. |
| An uninvited sign-up recorded as an access request, and the owner told | **Built and tested 2026-09-12.** One row per address; the owner is emailed on the first request only. Approving means adding the address to `ALLOWED_EMAILS`. |
| One account cannot see another's conversations or reach another's computer | **Built and tested 2026-09-12** (D-031). Every conversation query and hub event is scoped to the account; the shared-token daemon works for `OWNER_EMAIL`. API, store and hub tests cover lists, search, ids, edits, deletes, sends, stops, events and directory listing. Per-computer identity arrives with pairing (US2). |
| Proving a new person's email address, and resetting a forgotten password | **Built and tested 2026-09-12; real delivery proved 2026-09-13 on production.** Links are single-use, newest-wins, 30 minutes, stored hashed; completing one creates the account (or sets the password) and the session in one transaction. Mail goes over SMTP with TLS; `MAIL_TRANSPORT=log` prints it on a development machine and is refused with secure cookies. Registering `agent@sparstrow.com` at `https://app.sparstrow.com` sent a real SMTP message that landed in its inbox within seconds (read back through the Hostinger Email API), and its link completed the account; reusing the spent link was correctly refused. |
| Browser-approved machine login without copying a shared daemon secret | **Built 2026-09-13.** The browser mints an opaque ten-minute request; the local daemon spends it exactly once, receives a distinct 256-bit credential, writes it under `%LOCALAPPDATA%\sparstrowgen`, and waits for browser approval before it can open the daemon websocket. Only hashes are stored. The shared `DAEMON_TOKEN` route remains for running the daemon from source. |
| The browser recognising that the local component is installed or running | **Built 2026-09-13; proved on the owner's PC against production** (install, link, approval, providers, one Chat turn); sign-in persistence, a fresh Windows account and signing remain (G-32). One unsigned `sparstrowgen-setup.exe` (D-033) installs for the current Windows user, registers the `sparstrowgen://` handler and start at sign-in, and runs windowless. Opening the link claims the request by dialling out; after approval the daemon connects within about two seconds. The setup surface opens the link from an explicit action and, after eight seconds without a claim, offers retry and install help rather than asserting the component is absent — an unanswered link cannot be told apart from a slow or blocked one. |
| Storing a picture for an account | **Built and tested 2026-09-21 (D-048).** There is no object storage in this system. An avatar is bytes in Postgres, in its own `user_avatars` table rather than on `users` — that row is read with `SELECT *` on every authenticated request. The browser centre-crops to a square, scales to 256 and re-encodes as WebP before sending; the server caps it at 512KB and accepts PNG, JPEG and WebP only, never SVG. Measured on a real upload: a 900x500 PNG of 27.8KB stored as a 256px WebP of 4.8KB. Served `private` and `nosniff`, with the upload time as its cache key. **No bucket, no credential, no lifecycle** — it backs up with the database. |
| Disconnecting one computer | **Built and tested 2026-09-13.** Revocation is per-machine and closes that machine's live socket. Its next dial gets 403, so the daemon deletes the credential and stops instead of retrying. Other paired machines stay connected. |
| The server knowing a daemon's version and compatibility | **Built and tested 2026-09-13 (US3).** Hello carries `version`, `protocol` and `selfUpdates`; a daemon that sends none (0.1.x) is protocol 0 with no updater. The last version is kept on the machine row, so an offline computer still says what it runs. A computer below `MinDaemonProtocol` (0 today) is marked too old, Chat is told, and a send is refused with a message to update in Settings → Updates (`TestATooOldComputerIsToldToUpdateAndSentNoWork`). Chat and Settings → Updates were seen in that state 2026-09-14 on a local stack with the minimum raised (Unverified U-10). |
| Knowing whether sparstrowgen-managed agent work is active on this computer | **Verified in the current daemon.** It registers every running turn until that turn finishes or is stopped. That registry is the minimum reliable safety signal for blocking update activation. |
| Installed release channel, checked update, supervisor and rollback | **Built and tested 2026-09-13 (US3, D-034); no signing key since 2026-09-14 (D-035). A real self-update was proved on the owner's PC, 0.2.0 → 0.2.1 from published releases (Unverified U-7); so was an update with no key, 0.2.2 → 0.2.3 from releases GitHub Actions built (U-13); and so was rollback, 2026-09-14: a broken test build from a pre-release channel was put back after two minutes on the owner's PC (U-8).** GitHub Actions builds each release from a `daemon-v*` tag and publishes a manifest whose SHA-256 covers the installer. The daemon reads it over HTTPS, refuses an installer on any other site, and keeps a download only if it matches (`internal/release`, `cmd/daemon/update_test.go`). Installing waits until no turn is running, then an updater process started from the old version swaps the executables and puts the old one back if the new one has not reconnected within two minutes (`runUpdate` tests, with fake processes). Settings → Updates was walked in a browser against a local server and scripted computers. The executable itself is still unsigned for Windows (G-32). |

## Provider availability is three states, not two

Adopted from [Multica](../Reference/multica-main)'s `AgentAvailability` (`server/internal/service/agent_ready.go`):
not "ready or not", but *whether waiting is a plan*.

| State | Meaning | Example here |
|---|---|---|
| **Available** | Can take work now | `claude`, `codex`, `agy` — signed in, on PATH |
| **Waitable** | Not runnable right now, nothing broken — resolves itself | Daemon's machine is asleep |
| **Blocked** | Nothing will happen until a human acts | a CLI uninstalled or signed out |

**`Blocked` currently has no live example.** It used to be `gemini`, which was installed with no
account behind it. The owner has since ruled gemini out of the product entirely — see
[`Decisions.md`](Decisions.md) D-014 — so the state remains in the model but is now an unexercised
path. Keep it: a signed-out or uninstalled CLI is the same shape, and a UI must never silently omit
a blocked provider the way an unbuilt one is omitted; it should say why, the way Multica's
`RuntimeUnusableNotice` does. Check the design against the first real case rather than assuming it
still fits.

**A provider is the CLI we drive, never the vendor of the model.** `agy` is a router: `agy models`
returns Gemini, Claude *and* GPT-OSS models behind one CLI. Anything that treats a provider's
identity as the model's maker will be wrong for a third of the roster.

## What the agent CLIs emit

This is the hard ceiling on what a chat UI can show. We drive these CLIs; we don't control what
they report. Captured 2026-09-09 by running each with a trivial prompt and reading the real
output. What's still open after that capture is [`KnownGaps.md`](KnownGaps.md) G-4.

| | `claude` 2.1.278 | `codex` 0.154.0 | `agy` 1.2.0 |
|---|---|---|---|
| Non-interactive | `-p` *(verified)* | `codex exec` *(verified)* | `-p` *(verified)* |
| Streaming JSON | `--output-format stream-json --verbose` *(verified — `--verbose` is **required** with `-p`, undocumented in `--help`)* | `--json` JSONL *(verified — real stream captured)* | `--output-format stream-json` *(verified — real stream captured)* |
| Incremental text streaming | **yes** *(verified 2026-09-10 — 89 `content_block_delta` events averaging 8.1 chars for a four-sentence answer, the finest-grained of the three; needs `--include-partial-messages`)* | **no** *(verified — no delta event type exists; one whole `item.completed` per turn)* | **yes** *(verified — 93 chunks of ~25–35 chars for a 400-word answer)* |
| Config isolation | `--strict-mcp-config --setting-sources project` *(verified 2026-09-10 — 0 MCP servers, 72 skills down to 17. **NOT `--bare`**: it never reads OAuth)* | `--ignore-user-config` *(verified — zero MCP noise, auth still works)* | not needed in captures so far |
| Per-turn token usage | **verified** — see below | **verified** — `turn.completed.usage`: `input_tokens`, `cached_input_tokens`, `cache_write_input_tokens`, `output_tokens`, `reasoning_output_tokens`. The cache figures are **parts of** `input_tokens` and reasoning part of `output_tokens` (codex's own `TokenUsage`; B-56) | **verified** — `result.usage` and each `step_update.usage`: `input_tokens`, `output_tokens`, `thinking_tokens`, `cache_read_tokens`, `total_tokens` |
| Per-turn USD cost | **verified** — `total_cost_usd`, real dollar figure | no — tokens only | no — tokens only |
| Rate-limit signal | **verified** — `rate_limit_event`, see below. NOT parsed by the adapter yet (G-10) | not observed in this capture | not observed in this capture |
| Resume a session | `--resume <uuid>` *(verified flag)* | `codex exec resume <id>` *(verified flag)* | `--conversation <id>` *(verified flag)* |
| We choose the session id | `--session-id <uuid>` *(verified flag)* | no | no |
| Model override | `--model` *(verified)* | `-m` *(verified)* | `--model` *(verified)* |
| List available models | **yes, verified 2026-09-23 on 2.1.280** — a `list_models` **control request** over stream-json, no user message and nothing billed. It returns the same rows as the CLI's own `/model` picker, for the signed-in account: Opus 5.5, Sonnet 5, Fable 5.1, Haiku 4.5 and older ones. The reply nests the rows at `response.response.models` as `value` / `resolvedModel` / `displayName`. **Signed out, it answers with built-in defaults**, which lag the account (Opus 5 where the account has Opus 5.5). The daemon asks again every 30 minutes (B-52) | **no** — names scraped from the binary, configured one read from `~/.codex/config.toml`; an invalid model returns a 400 naming no alternatives, and the valid set is **account-dependent** (`"not supported when using Codex with a ChatGPT account"`) | `agy models` *(verified, returns id + label)* |
| Model carries reasoning effort | not observed | `model_reasoning_effort` in config, separate from the model | **in the model id** *(verified — `gemini-3.1-pro-high` and `-low` are distinct models, not one model with a setting)* |
| Messages per turn | **several** *(verified 2026-09-10 — one turn was four `assistant` events: thinking, prose, `tool_use`, answer. Only `text_delta` carries `.text`, so thinking and tool arguments cannot leak in)* | **several** *(verified — two `item.completed` `agent_message` items in one turn)* | **one** *(verified 2026-09-10 — `result.response` is a single authoritative string, byte-identical to the concatenated deltas)* |
| Session is scoped to a directory | **yes** *(verified 2026-09-10 — `--resume <id>` from a different cwd answers `No conversation found with session ID`, and the turn fails in about a second. Moving a conversation's folder therefore has to drop the session)* | not tested | not tested |

**All three real providers report usage; `claude` also reports real dollar cost.** This overturns
what this file originally said. `claude`, `codex` and `agy` each stream a structured usage object;
the fields differ but all three exist and are real.

### The rate-limit signal — the mechanism this entire product depends on

**Verified, from a real `claude -p` capture, 2026-09-09.** Every turn emits a
`rate_limit_event` alongside the assistant message:

```json
{"type":"rate_limit_event","rate_limit_info":{
  "status":"allowed","resetsAt":1789009200,"rateLimitType":"five_hour",
  "overageStatus":"rejected","overageDisabledReason":"org_level_disabled","isUsingOverage":false
}}
```

`resetsAt` is a Unix timestamp for when the current window resets. `rateLimitType` names the
window (`"five_hour"` observed). This is precisely the signal a provider-switch offer is built on
— **once a design needs it, this is real and buildable for `claude`.**

**Only `status: "allowed"` has been observed.** We have not seen what `status` (or the rest of the
payload) looks like when a limit is actually hit, because the capture succeeded well inside limits.
Do not assume the blocked value or shape without a real capture — see G-4.

**`codex` and `agy` emitted no equivalent event in their captures.** Unknown whether either has
one, reports it differently (e.g. only as a non-zero exit or an error message when truly
exhausted), or doesn't expose it at all. This is the single most important remaining question for
this product's core feature, and it can only be answered by hitting a real limit or finding
provider documentation — it cannot be forced in a trivial test.

### Real event shapes captured

**`claude`** (verified, real capture, real account, plain `stream-json` — not
`--include-partial-messages`): `{"type":"system","subtype":"init",...}` →
`{"type":"assistant","message":{...,"usage":{...}}}` (one full message per turn in this mode) →
`{"type":"rate_limit_event",...}` → `{"type":"result","subtype":"success","total_cost_usd":...,
"usage":{...},"modelUsage":{"<model>":{...,"contextWindow":200000,"maxOutputTokens":32000}},
"permission_denials":[]}`.

**`codex exec`** (verified, real capture): `{"type":"thread.started","thread_id":...}` →
`{"type":"turn.started"}` → `{"type":"item.completed","item":{"type":"agent_message","text":...}}`
→ `{"type":"turn.completed","usage":{...}}`.

**`agy`** (verified, real capture): `{"event":"init",...}` → repeated
`{"event":"step_update","step_update":{"step_index":n,"state":"ACTIVE"|"DONE","text_delta":...}}`
→ `{"event":"result","result":{"status":"SUCCESS","response":...,"usage":{...}}}`. Token deltas
arrive on the `state:"ACTIVE"` updates; the final usage total is on `state:"DONE"` and again on
`result`.

### Incremental streaming differs per provider — and the design must absorb it

**Verified 2026-09-09 by capturing a 400-word answer from each.** This is not a uniform capability,
and a chat design that assumes it is will look broken on one provider:

- **`agy` streams.** 93 `step_update` events with `state:"ACTIVE"`, each carrying a `text_delta` of
  roughly 25–35 characters. Word-group granularity, not strictly per-token, but more than enough
  for a live "typing" feel. *(An earlier short capture showed a single chunk — that was the answer
  being too small to chunk, not the absence of streaming.)*
- **`codex` does not stream.** `--json` is its only stream flag and there is no delta event type at
  all: the turn emits `thread.started` → `turn.started` → one whole `item.completed` → `turn.completed`.
  The reply appears all at once, however long it took.
- **`claude` is unverified.** Plain `stream-json` gives one complete message per turn;
  `--include-partial-messages` is documented to give deltas but has not been captured. See G-4.

**The design consequence, and it is a real one:** the same conversation can have one provider
typing smoothly and the next sitting silent for twenty seconds before a wall of text lands. A
design whose only "working" signal is text appearing will read as *frozen* on `codex`. Whatever
indicates "the agent is working" must be independent of text arriving — and it must be present in
the design from the first draft, not retrofitted when codex is wired up.

`gemini` is out of the product entirely (D-014). Nothing here covers it and nothing should.

### Files the owner drops into a conversation (2026-09-11)

Checked because nothing here covered attachments, and the answer decides what the feature *is*.

| | `claude` 2.1.278 | `codex` 0.154.0 | `agy` 1.2.0 |
|---|---|---|---|
| A flag that attaches a file to the prompt | **no** — `--file` takes Claude's own file-API ids (`file_abc:doc.txt`), not local paths *(verified from `--help`)* | **`-i, --image <FILE>...`**, on `exec` **and** `exec resume` *(verified flag, images only)* | **no** — its `-i` is `--prompt-interactive`, unrelated *(verified from `--help`)* |
| Reads a file from disk when the prompt names the path | assumed — its read tool handles images, not re-checked here (the token in the checking shell had expired) | **yes, including images** *(verified 2026-09-11 — a PNG containing "SECRET CODE: PELICAN-7429" and "count of rows: 314" was read from cwd and both values returned, with no attachment flag and no vision plumbing from us)* | **no — denied before it tries**, see [`Bugs.md`](Bugs.md) B-11 |

**The consequence for design.** Attaching a file *to the model* is not deliverable across providers
— one of three has a flag and only for images. Putting the file **on the machine, inside the folder
the conversation already runs in, and naming it in the prompt** is deliverable, for any file type,
using the agents' own tools — and it is strictly better than an attachment: the file stays readable
on later turns, survives a provider switch because it lives in the folder rather than in a session,
and can be grepped and diffed rather than only looked at.

**Except on `agy`, which can use no tools at all** (B-11). A design that assumes every provider can
read what was dropped is not deliverable today; one that states plainly what the current provider
can do with it is.

**Only the daemon can write the file**, per D-020 — and it must be written before the turn starts,
because the agent has no way to call back for it. That is the one place this differs from
[Multica](../Reference/multica-main), which ships a CLI on the machine and can therefore tell the
agent to fetch the bytes itself (`multica attachment download <id>`); its prompt deliberately
carries the id and filename rather than a URL, because a signed URL can expire before the agent
gets to it (`server/internal/daemon/prompt.go`). Materialising first has no such failure mode.

### Agent activity — what each CLI says about the work it does (2026-09-24)

**Verified** from one real turn per provider, with the daemon's exact arguments and a clean
environment, in a scratch folder. The prompt asked each agent to read a file, append a line to it,
run `git --version`, search the web, and answer in one sentence.

| | `claude` 2.1.280 | `codex` 0.154.0 | `agy` 1.2.3 |
|---|---|---|---|
| Live, step by step | **yes** — `content_block_start` for each `tool_use` as it begins, the arguments streamed as `input_json_delta`, the result in a following `user` message as `tool_result` | **yes** — `item.started` then `item.completed` per item | **yes** — `step_update` with `state` `ACTIVE` → `DONE` / `ERROR` and `duration_seconds` |
| Thinking | a `thinking` block whose **text is empty**, only a signature. That it thought, and when, is known; what it thought is not | none seen (`reasoning_output_tokens: 0`) | a `thinking_tokens` count only |
| Read a file | `Read` with `file_path`; the result is the content | refused (see below) | `view_file` with `AbsolutePath` |
| Edit a file | `Edit` with `file_path`, `old_string`, `new_string` — so lines added and removed can be counted | refused | refused |
| Run a command | `PowerShell` (on Windows) with `command` and `description`; the result is the output | refused | `run_command` with `CommandLine`, refused |
| Search the web | `WebSearch` with `query`, refused | `web_search` item with `query` — **no sources** | `search_web` with `query` — **no sources** |
| Narration between steps | `text` blocks between tool calls ("The edit was denied. Let me retry it…") | `agent_message` items, several per turn | the one `result.response` |
| Refusals | `tool_result` with `is_error: true` and the reason; `system:permission_denied`; `result.permission_denials` lists each | `CreateProcess … rejected: blocked by policy` on **stderr only** | `step_update` `state: ERROR` and `result.denied_actions` |

**What cannot be shown, so must not be designed:** thinking *text* from any of the three, and the
sources a web search read. Only claude's result contains search results at all, and that was
refused in this capture, so its shape is unverified.

**Under the daemon's current arguments, the agents mostly cannot act.** claude read the file and
ran the command, but was refused the edit (twice) and the web search. codex could not read, write
or run anything and could only search. agy's command was auto-denied ("a tool required the
`command` permission that headless mode cannot prompt for"). No file was changed. What the agents
should be allowed to do is the owner's decision: docs/Later.md L-33.

agy also looked in its own scratch folder rather than the conversation's; fixed in B-55.

### What a turn changed: diffs, from the daemon rather than the agent (2026-09-24)

**No agent reports a diff we can use for all three.** claude's `Edit` carries the old and new text
but no line numbers. codex and agy were refused every edit, so their shapes are unseen. And a file
changed by a *command* (a formatter, `sed`, a code generator) is reported by nobody.

**The daemon can record it itself, the same way for every agent, verified on git 2.53 (Windows).**
It keeps a separate git store of its own for the conversation, outside the folder, and points it at
the folder with `--work-tree`, using an index file of its own. Before a step and after it, `add -A`
plus `write-tree` records a snapshot. `diff-tree -p` between two snapshots gives a unified diff with
real line numbers, and `--numstat` gives the +/− counts. In a scratch test: an edited file, a new file
and a deleted file were all reported correctly. The folder's own `.gitignore` was honoured
(`node_modules/` stayed out). The folder's own git repository was left untouched, including what the
user had staged, and no stash was created.

| What the design can show | Deliverable |
|---|---|
| Per turn: which files were created, edited or deleted, with +/− counts | **yes** |
| The diff itself, with old and new line numbers | **yes** |
| Per step (which edit or command changed what) | **yes**, a snapshot after each step. Steps that run in parallel share one diff |
| Live, while the turn runs | **yes**, at each step's end |
| The whole conversation's changes ("all changes") | **yes**, from the snapshot before its first turn to the latest |
| The unchanged lines between hunks, expanded | not designed; the snapshots hold them, but storing them means storing whole files |
| Changes the person made between turns | **no**: each turn's snapshot starts afresh, so these are not the agent's and are not shown |

**Limits to design for:** the computer needs git. Claude Code on Windows requires Git for Windows,
but a computer with only codex or agy may not have it, so "changes are not recorded" is a real state.
A folder with no `.gitignore` and a huge tree (`D:\`) needs a cap. The diff is sent to the server and
kept with the turn, bounded like the raw exchange, so it survives a refresh and shows on another
device.

### The raw exchange — what one turn's record can hold (2026-09-23)

For [the raw-exchange spec](specs/2026-09-23-raw-exchange.md). **Verified by reading the daemon and
the captured streams** in `server/internal/agent/testdata`; no new CLI run. **Since D-053 all of the
first three rows are kept**, per turn; before it, the "Before D-053" column is what survived.

| | What exists | Before D-053 |
|---|---|---|
| What we send | All of it is ours: the daemon builds the prompt (the catch-up framing is `buildPrompt` in `cmd/daemon/main.go`), the arguments, the folder and the resume id | only the user's message, and a replay marker with a count and tokens |
| The CLI's stdout | every event, all three providers | parsed for text, session id, usage, cost and limits; the rest was discarded |
| The CLI's stderr | codex's refusals appear **only** here (Agent activity, above) | **never read**: no adapter attached stderr, so it went to the null device |
| What the CLI loaded by itself | claude's `system:init`: `cwd`, `model`, `claude_code_version`, `permissionMode`, `tools`, `skills`, `slash_commands`, `agents`, `mcp_servers`, `plugins`, `output_style`. codex's `thread.started`: the thread id only. agy's `init`: conversation id and `cwd` only | nothing |
| How much it loaded | usage, all three (table above). The captured claude turn: 4 new input tokens, 30,219 read from cache, 9,177 written to it — about 39k tokens of claude's own context behind a short message | the total only |
| The text of a CLI's built-in instructions | **not emitted by any of the three** on their output — but **each keeps it in its own session files on the computer**; see the next section | — |
| Thinking text | **depends on the version.** The 2.1.90 capture (Sonnet 4.6) streams it as `thinking_delta` text; the 2.1.280 capture sent the block empty, signature only. codex and agy: none seen | — |

**Size.** The captured claude turn is 22.5 KB of events for a two-message answer, mostly
`--include-partial-messages` deltas. A turn whose tools read files carries those files in full.

### What each agent loaded, word for word — from its own session files (2026-09-24)

Asked by the owner: the tokens an agent reads beyond your message are its own instructions, skills,
tools and rules, and he wants to see them, to turn off what he does not want fed. **None of the three
prints them. All three write them to disk**, in their own session store on the computer the daemon
already runs on. **Verified 2026-09-24 by reading the files the test turns of U-32 left behind** (and,
for agy, a recent conversation); no CLI was run.

| | Where | What is in it | What is not |
|---|---|---|---|
| **claude** 2.1.280 | `~/.claude/projects/<folder, non-alphanumerics as ->/<session id>.jsonl`, one JSON object per line | a `prompt_snapshot` attachment: the **whole system prompt** (15 sections for the test turn — the largest, 12,971 characters, is its "auto memory" instructions) and **every tool's full definition** (14; Bash alone is 22 KB). Also `instructions` (each CLAUDE.md with its **path**, Project or User, and content), `nested_memory` (CLAUDE.md files in subfolders), `skill_listing` (the skills offered, with descriptions), `mcp_instructions_delta` (each MCP server's instructions), `deferred_tools_delta` (tools held back until searched for), `agent_listing_delta`, `invoked_skills` | a skill's file path |
| **codex** 0.154 | `$CODEX_HOME/sessions/YYYY/MM/DD/rollout-…-<thread id>.jsonl`, JSON lines | `session_meta.base_instructions` (**the system prompt**, 17,730 characters for GPT-5.6 Sol), developer messages: `<skills_instructions>` (42 skills with descriptions **and file paths**, 22,695 characters), the agent-role and multi-agent instructions, `<recommended_plugins>`; a `world_state` (AGENTS.md, host skills, permissions, personality); `turn_context` (sandbox `read-only`, approval `never`, model) | **tool definitions** (only mentioned by name in calls) |
| **agy** 1.2.3 | `~/.gemini/antigravity-cli/conversations/<conversation id>.db`, SQLite; the prompt is a **protobuf blob** in `gen_metadata` | the whole generation input: an `<identity>` system prompt (28,997 characters), `<user_rules>` (11,875 characters), tool definitions as JSON schemas, `<artifacts>` instructions, and the conversation | a published schema: the text is readable, but which piece is which is read off tags like `<identity>` and `<user_rules>`; where the rules came from is not recorded |

**What this makes possible:** after each turn the daemon reads that turn's entry from the agent's own
store, by the session id it already has, and adds it to the turn's record. So each turn can show
every piece that was fed, its size, and, for claude's instruction files and codex's skills, which
file it came from.

**What it costs, and the risk:**
- **These are private files, not an interface.** Any CLI update can move or reshape them, as `claude`
  and `codex` do often. It must fail visibly ("could not read claude's session file"), never quietly
  show nothing.
- **Size.** claude's snapshot is about 127 KB per turn, agy's about 116 KB, codex's about 60 KB. They
  are mostly the same from turn to turn, so they are worth storing once and pointing at.
- **They hold his own rules and memory** (CLAUDE.md, GEMINI.md, claude's memory instructions), which
  would then sit on the server with the rest of the record (G-48).
- **agy's store is written while a turn runs,** so it is copied before it is read, never opened in place.

## Real caveats found while capturing (2026-09-09)

- **`codex exec` loads the owner's global `CODEX_HOME` config by default — and `--ignore-user-config`
  fixes it.** An unscoped run emitted `AuthRequired` stderr noise for Supabase and GitHub Copilot
  MCP servers unrelated to this app. `codex exec --json --ignore-user-config` was re-captured
  2026-09-09 and produced **zero** MCP noise while still authenticating (the flag skips
  `config.toml` but keeps using `CODEX_HOME` for auth). **The daemon's `codex` adapter passes
  `--ignore-user-config`** — codex's equivalent of scoping claude. Not optional, for the same
  scope-leak reason.
- **`claude -p` inherits the entire personal Claude Code environment, and the tools actually work
  — SOLVED 2026-09-10.** Unscoped, the `system.init` payload showed `clockify`, `square`, and
  `shadcn` MCP servers all `"status":"connected"` — not merely listed, genuinely reachable. A
  conversation that only asked for `"OK"` had *working* access to the owner's time-tracking and
  invoicing tools. That is a scope leak, not a performance concern.

  **`--bare` is the wrong fix and was recorded here in error.** It never reads OAuth or the keychain
  (its own `--help` says so), and this account signs in with a subscription rather than an API key,
  so `claude --bare -p` returns `Not logged in`. The working recipe, captured 2026-09-10 with every
  `CLAUDE_*` and `ANTHROPIC_*` variable scrubbed from the environment:

  ```
  claude -p --output-format stream-json --verbose --strict-mcp-config --setting-sources project
  ```

  `"apiKeySource":"none"` and the turn still ran, so OAuth works. `"mcp_servers":[]` — zero
  connected servers. `--setting-sources project` additionally cuts inherited skills from 72 to 17.
  This is what [Multica](../Reference/multica-main) does (`server/pkg/agent/claude.go`), which is
  what prompted the recheck.
- **A nested `CLAUDE_*` environment makes `claude -p` hang.** Two runs from an agent shell were
  killed at 120s with stdin both attached and closed. The same command with `CLAUDE_*` and
  `ANTHROPIC_*` scrubbed returns in seconds. `CLAUDE_CODE_MESSAGING_SOCKET`, `CLAUDECODE=1` and
  friends are inherited by any child process. **The daemon must scrub its environment before
  spawning any agent CLI**, or it will hang the moment anyone runs it from inside a Claude session.
- **`agy`'s tool list is large** — browser control, subagents, image generation, scheduling, and
  more are all available by default (`init.tools`, real capture). It is closer to a full autonomous
  agent than a text generator. Any chat surface that shows "what the agent can do" needs to reflect
  that `agy` starts from a much bigger toolbox than `claude -p` or `codex exec` do by default.

## Do not design these yet

- **A live model picker populated for every provider.** Only `agy` lists its models. For the others
  the list is curated by us and will drift from what the account can actually use — and for `codex`
  the valid set depends on the account behind it, so even a correct list is correct per-account.
  A picker is fine; treating it as authoritative is not. See [`KnownGaps.md`](KnownGaps.md) G-8.
- **Anything assuming a shared session across providers.** No CLI can resume another's session.
  Switching provider means replaying history into a fresh session — so a design implying one
  continuous thread with the provider is a lie the backend cannot make true. The *conversation* is
  continuous; the provider session is not.
- **A chat surface whose only sign of progress is text appearing.** `codex` emits nothing until the
  whole answer lands. Progress must have its own affordance.
- **Instant response to a keystroke that requires the machine.** The round trip is too long.
- **Anything requiring the agent to control the desktop** — mouse, keyboard, screen pixels. Not
  built, and deliberately not planned. See [`Later.md`](Later.md) L-1.

## Safe to design against

- **Streaming assistant text — but only on some providers.** Verified incremental on `agy`,
  verified *not* incremental on `codex`, unverified on `claude`. Design the streaming case, and
  design a working-but-silent case that doesn't look frozen. See the streaming section above.
- **A persistent conversation that outlives any single provider session**, because our database is
  the transcript of record.
- **Switching provider mid-conversation**, with the new provider replayed the history it hasn't
  seen.
- **Per-turn usage for all three real providers**, and real USD cost specifically for `claude`.
  Design a usage display now; `claude` can show a dollar figure, `codex`/`agy` show tokens only.
- **A rate-limit indicator for `claude`**, using `rate_limit_event`'s `resetsAt` and
  `rateLimitType`. Design the "provider is close to its limit" state around this — but not yet the
  "provider is over its limit" state, since only `status: "allowed"` has been observed. `codex` and
  `agy` have no verified equivalent; a design needing this for them is not yet deliverable.
- **Per-run metadata**: which provider, which model, when it started and ended, whether it
  succeeded or failed.
- **Provider and model discovery from the machine** — the daemon can probe what's installed and
  report it, including a `Blocked` state with a reason, per the three-state model above.
- **Daemon online/offline state**, and which directories are registered.
- **Stopping a turn that is already running, and keeping what it had said.** Verified 2026-09-11 on
  `claude` and `codex`. The daemon owns each CLI's whole process tree (a Windows Job Object, or a
  process group elsewhere), so a stop takes the agent's own subprocesses with it: verified against a
  real turn where `claude` had spawned `bash -c 'sleep 180'`, and both were gone immediately
  afterwards. The process-group path is verified too, on Linux under `make test-linux`. Text that
  had already arrived is kept and the turn is recorded as stopped rather than failed.
- **A turn always ends.** Verified 2026-09-11. It ends because the provider finished, because it
  failed, because the owner stopped it, or because the machine disconnected — and in every case the
  entry is closed out, the partial text kept, and the composer released. A machine that stays
  connected but goes silent is covered too, by an inactivity watchdog in the daemon — 15 minutes of
  no output at all, tunable with `TURN_IDLE_TIMEOUT`. Note what that does NOT promise: it is
  silence, not duration, so a turn is never ended for taking a long time while it is still
  producing.
- **Partial output survives a stop on `codex` too, despite it not streaming.** Verified 2026-09-11,
  and it corrects a reasonable-sounding assumption: `Streams: false` means codex emits no
  incremental *deltas*, NOT that it produces nothing until the end. It completes whole
  `agent_message` items during a turn, and a stop keeps every one it finished — a codex turn stopped
  after 6s kept 471 characters the surface had never displayed, because there were no deltas to
  display them with. So "does not stream" and "has nothing to keep" are different claims, and only
  the first is true.

## Keeping this honest

When a design asks for something not listed here, there are three legitimate answers, and
"probably fine" is not one of them:

1. **Check it.** Run the CLI, capture the stream, and add a verified row. Cheapest option, and
   usually minutes.
2. **Design around it.** Change the design so it needs only what's deliverable.
3. **Record it as a gap.** If the design genuinely needs it and it isn't deliverable yet, that goes
   to [`Later.md`](Later.md) with a trigger, and the design ships without that piece.

## Authentication, per provider (2026-09-10)

Learned the hard way, by running all three from a spawned process rather than a terminal.

| | How it authenticates from a spawned process |
|---|---|
| `claude` | **`CLAUDE_CODE_OAUTH_TOKEN`**, from `claude setup-token`. The credentials in `~/.claude/.credentials.json` hold an access token that only the desktop app refreshes, so a plain child process eventually meets *"OAuth access token has expired"* and cannot recover. The variable is exempted from the daemon's env scrubbing on purpose. |
| `codex` | `CODEX_HOME` credentials, which `--ignore-user-config` deliberately keeps using. Nothing extra needed. |
| `agy` | Its own stored login. Nothing extra needed. |

**Read `is_error`, never `subtype`.** A failed claude turn reports
`"subtype":"success"` alongside `"is_error":true`. Checking the subtype turns a
total authentication failure into an apparent success — which is exactly the
mistake made on the first pass here, and it cost a wrong claim to the owner.

**Environment changes need a new process.** A token set through the Windows
Environment Variables dialog is not inherited by anything already running. If
claude fails to authenticate while codex and agy work, check that first: it is
far more likely than a bad token.
