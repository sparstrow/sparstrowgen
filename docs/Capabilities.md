# Capabilities — what the backend can actually deliver

**Read this before designing anything.** Design-driven development only works if the design stays
inside what the backend can produce. A screen showing data no provider emits is waste — it looks
finished, the owner approves it, and then the backend can't serve it.

This file is the feasibility surface. It is not a plan and not a roadmap: it says what is
*possible*, not what is scheduled.

Mark every claim as **verified** (we ran it and saw real output) or **assumed** (looks true from
documentation or flags, not yet proven). Downgrade nothing silently — if something assumed turns
out false, fix it here in the same change that discovers it.

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

## Provider availability is three states, not two

Adopted from [Multica](../Reference/multica-main)'s `AgentAvailability` (`server/internal/service/agent_ready.go`):
not "ready or not", but *whether waiting is a plan*.

| State | Meaning | Example here |
|---|---|---|
| **Available** | Can take work now | `claude`, `codex`, `agy` — signed in, on PATH |
| **Waitable** | Not runnable right now, nothing broken — resolves itself | Daemon's machine is asleep |
| **Blocked** | Nothing will happen until a human acts | `gemini` — installed, **no account access** |

**`gemini` is `Blocked`, not `not built`.** The CLI is present (`gemini` 0.49.0 on PATH, confirmed
2026-09-10), but the owner has no account signed into it. Building an adapter for it is pointless
until that changes — a UI must never silently omit a blocked provider the way an unbuilt one is
omitted; it should say why, the way Multica's `RuntimeUnusableNotice` does. Tracked as
[`Later.md`](Later.md) L-6. Do not build a `gemini` adapter against this entry — build it against a
real capture once he has access.

## What the agent CLIs emit

This is the hard ceiling on what a chat UI can show. We drive these CLIs; we don't control what
they report. Captured 2026-09-10 by running each with a trivial prompt and reading the real
output. What's still open after that capture is [`KnownGaps.md`](KnownGaps.md) G-4.

| | `claude` 2.1.90 | `codex` 0.153.4 | `agy` 1.1.27 |
|---|---|---|---|
| Non-interactive | `-p` *(verified)* | `codex exec` *(verified)* | `-p` *(verified)* |
| Streaming JSON | `--output-format stream-json --verbose` *(verified — `--verbose` is **required** with `-p`, undocumented in `--help`)* | `--json` JSONL *(verified — real stream captured)* | `--output-format stream-json` *(verified — real stream captured)* |
| Per-turn token usage | **verified** — see below | **verified** — `turn.completed.usage`: `input_tokens`, `cached_input_tokens`, `cache_write_input_tokens`, `output_tokens`, `reasoning_output_tokens` | **verified** — `result.usage` and each `step_update.usage`: `input_tokens`, `output_tokens`, `thinking_tokens`, `cache_read_tokens`, `total_tokens` |
| Per-turn USD cost | **verified** — `total_cost_usd`, real dollar figure | no — tokens only | no — tokens only |
| Rate-limit signal | **verified** — `rate_limit_event`, see below | not observed in this capture | not observed in this capture |
| Resume a session | `--resume <uuid>` *(verified flag)* | `codex exec resume <id>` *(verified flag)* | `--conversation <id>` *(verified flag)* |
| We choose the session id | `--session-id <uuid>` *(verified flag)* | no | no |
| Model override | `--model` *(verified)* | `-m` *(verified)* | `--model` *(verified)* |
| List available models | no — curated list | no — read config | `agy models` *(verified, returns id + label)* |

**All three real providers report usage; `claude` also reports real dollar cost.** This overturns
what this file originally said. `claude`, `codex` and `agy` each stream a structured usage object;
the fields differ but all three exist and are real.

### The rate-limit signal — the mechanism this entire product depends on

**Verified, from a real `claude -p` capture, 2026-09-10.** Every turn emits a
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

**Token-by-token streaming (deltas) is still unverified for all three.** Every capture so far used
the non-partial stream mode — each shows one complete message per turn, not incremental text. The
"typing" feel a chat UI wants needs `claude --include-partial-messages` (and codex/agy's
equivalents, if any) captured separately. See G-4 — this affects rendering smoothness, not data
modeling, so it is lower priority than the rate-limit question above.

`gemini` 0.49.0 is installed but blocked (no account) — see above. Not captured, and not worth
capturing until it is.

## Real caveats found while capturing (2026-09-10)

- **`codex exec` loads the owner's global `CODEX_HOME` config**, including MCP servers configured
  for unrelated projects. The capture emitted `AuthRequired` stderr noise for Supabase and GitHub
  Copilot MCP servers that have nothing to do with this app. **The daemon must run `codex` with an
  isolated or minimal config** — a scoped `CODEX_HOME`, or the equivalent of `claude`'s `--bare` —
  or every run leaks unrelated auth errors and possibly unrelated tool calls into a chat turn.
- **`claude -p` inherits the entire personal Claude Code environment, and the tools actually work.**
  Two captures confirm this, and the second is worse than the first: from a real terminal, the
  `system.init` payload showed `clockify`, `square`, and `shadcn` MCP servers all
  `"status":"connected"` — not merely listed, genuinely reachable. A conversation that only asked
  for `"OK"` had *working* access to the owner's time-tracking and invoicing tools. **This is not
  a performance concern, it is a scope leak.** `--bare` on every spawned `claude` call is mandatory
  for the daemon's adapter, not an optimization — see [`KnownGaps.md`](KnownGaps.md) G-3.
- **`agy`'s tool list is large** — browser control, subagents, image generation, scheduling, and
  more are all available by default (`init.tools`, real capture). It is closer to a full autonomous
  agent than a text generator. Any chat surface that shows "what the agent can do" needs to reflect
  that `agy` starts from a much bigger toolbox than `claude -p` or `codex exec` do by default.

## Do not design these yet

- **A live model picker populated for every provider.** Only `agy` lists its models. For the others
  the list is curated by us and will drift from what the account can actually use.
- **Anything assuming a shared session across providers.** No CLI can resume another's session.
  Switching provider means replaying history into a fresh session — so a design implying one
  continuous thread with the provider is a lie the backend cannot make true. The *conversation* is
  continuous; the provider session is not.
- **Instant response to a keystroke that requires the machine.** The round trip is too long.
- **Anything requiring the agent to control the desktop** — mouse, keyboard, screen pixels. Not
  built, and deliberately not planned. See [`Later.md`](Later.md) L-1.
- **A `gemini` surface of any kind.** Blocked, not merely unbuilt — see above.

## Safe to design against

- **Streaming assistant text, token by token.** This is the core interaction and it works.
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

## Keeping this honest

When a design asks for something not listed here, there are three legitimate answers, and
"probably fine" is not one of them:

1. **Check it.** Run the CLI, capture the stream, and add a verified row. Cheapest option, and
   usually minutes.
2. **Design around it.** Change the design so it needs only what's deliverable.
3. **Record it as a gap.** If the design genuinely needs it and it isn't deliverable yet, that goes
   to [`Later.md`](Later.md) with a trigger, and the design ships without that piece.
