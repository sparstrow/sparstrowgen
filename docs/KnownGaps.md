# Known Gaps

**What you should know before trusting an area of this system.** Read it before relying on
something, and before claiming it works.

Two kinds of entry live here, and both answer the same question — *how much can the next agent
take on faith?*

| Kind | What it is |
|---|---|
| `unproved` | We built it, but couldn't fully prove it works — or proved it works only within limits |
| `caveat` | Something noticed in passing and deliberately left alone: fragile, surprising, half-finished, or true-but-unobvious |

Neither is a bug report. If something is actually behaving **wrong**, it goes in [`Bugs.md`](Bugs.md). If it is a
question, a parked decision, or an idea, it goes in [`Later.md`](Later.md).

## When to write one

**In the same turn it surfaces**, not later — whether it came from your own work or from something
you noticed while doing something else.

- Ticked a checklist item on weaker evidence than it asked for → say so where you ticked it, *and*
  open an `unproved` entry here.
- Noticed something odd and didn't act on it because it was out of scope → open a `caveat`. Going
  back to fix it is a separate decision; recording it is not optional.

A caveat that lives only in a chat message does not exist. The next session does not read chat.

## When to close one

**Delete the entry** and say where the proof lives, or which change fixed it. The length of this
file is a real signal — a gap lingering because closing it was inconvenient is exactly the failure
this register exists to prevent.

Ids are never reused.

## Format

```
## G-n — <the claim, or the thing noticed>
**Kind:** unproved | caveat
**Raised:** <YYYY-MM-DD>, <what was being done at the time>

<What is verified and what is not — be precise about the boundary. For a caveat: what you saw,
where (file:line), and why you left it. "The platform won't emit the signal" and "nobody got round
to it" are different situations and the reader needs to know which.>

- **If wrong:** <the cost if the assumption doesn't hold. "Cosmetic and self-correcting" is a
  legitimate answer — be honest in both directions.>
- **Clears when:** <the concrete thing that closes it — an action someone can take, not "when we
  have time".>
```

---

## G-4 — What a *hit* rate limit looks like, on any provider

**Kind:** unproved
**Raised:** 2026-09-09, after closing G-1. **Narrowed:** 2026-09-09, streaming half split out to G-5.

Two related unknowns, neither closable by running anything:

1. **`claude`'s `rate_limit_event` has only ever been seen as `"status":"allowed"`.** We do not
   know the blocked value, or whether the payload changes shape when a limit is exceeded.
2. **`codex` and `agy` emitted no rate-limit signal at all** in any capture. Unknown whether they
   have one, surface it only as an error or non-zero exit when truly exhausted, or never expose it.

This is the feasibility boundary under the product's headline feature. It does **not** block the
feature: switching provider mid-conversation is user-initiated and works regardless. It blocks the
*automatic* version — "you're out on claude, switch to codex?" — and the "over your limit" UI state.

- **If wrong:** an "over limit" state gets designed against a guess and needs correcting the first
  time a real limit is hit. Worse for codex/agy: a design promising limit awareness for all three
  providers would be undeliverable for two of them.
- **Clears when:** a provider is used enough to actually hit a limit — capture the stream when it
  happens, it is the only cheap opportunity — or provider documentation describes the payload.

## G-5 — `claude`'s incremental streaming is unverified

**Kind:** unproved
**Raised:** 2026-09-09, split from G-4 once codex and agy were verified.

`agy` streams (93 delta chunks for a 400-word answer) and `codex` provably does not (no delta event
type exists in `--json`). `claude` sits between them unverified: plain `stream-json` gives one
message per turn, and `--include-partial-messages` has never been captured — the flag is already in
`blueprint.yaml`'s `print` string on the strength of documentation alone.

**Two captures were run on 2026-09-10 and neither settles it.** Both used
`--include-partial-messages` and both returned **zero** `stream_event` /
`content_block_delta` entries — but both also 401'd ten times and never authenticated (G-9), so the
request never got far enough to stream anything. Absence of deltas from a turn that never reached
the API is not evidence about streaming.

The adapter therefore reads deltas if they appear and never depends on them, and the surface reports
`streams: false` for claude so it does not promise motion it may not deliver.

- **If wrong:** claude renders per-message like codex instead of per-token. A polish downgrade, not
  a broken feature — and the design already has to tolerate a non-streaming provider because of
  codex, so nothing designed against this assumption gets thrown away.
- **Clears when:** G-9 is fixed and one capture with `--include-partial-messages` completes a turn
  whose `is_error` is false — then count the `content_block_delta` entries. Until claude can
  authenticate, this cannot be answered at all.

## G-7 — Conversation search runs in memory over everything loaded

**Kind:** caveat
**Raised:** 2026-09-10, feedback round item 6.

`apps/web/lib/conversation-search.ts` scans every conversation and every message on each keystroke.
That is correct for a prototype holding seven conversations and wrong the moment transcripts are
real: the client would have to hold every message of every conversation to search them, which is
exactly what our own database exists to avoid.

The search is deliberately shaped so the move is mechanical — one function, taking a list and a
query, returning matches with an excerpt. The Postgres version answers the same shape.

- **If wrong:** nothing today. It degrades gradually with transcript volume rather than failing,
  which is the risk — it will keep seeming fine while quietly loading more than it should.
- **Clears when:** search is served by a query against Postgres and the client no longer needs the
  full transcript set in memory to run it.

## G-8 — Only `agy` can enumerate its own models

**Kind:** caveat
**Raised:** 2026-09-10, feedback round item 1.

The three CLIs are not equal here, and the model lists in the app come from three different grades
of evidence:

| Provider | How the list was obtained | Grade |
|---|---|---|
| `agy` | `agy models` — prints ids and labels | **verified**, re-runnable |
| `claude` | Each documented alias run once, reading `model` back out of `system.init`: `opus` → `claude-opus-4-6`, `sonnet` → `claude-sonnet-4-6`, `haiku` → `claude-haiku-4-5-20251001` | **verified for the three aliases**; whether other ids exist is unknown |
| `codex` | No list command. Names scraped from the shipped binary's string table, cross-checked against `model = "gpt-5.6-sol"` in `~/.codex/config.toml` | **partial** — the configured one is certain, the siblings are inferred |

**`claude` does have a real discovery mechanism — our CLI is just too old for it.** A `list_models`
**control request** over the stream-json control protocol returns the catalogue without sending a
user message, so nothing is billed:

```
echo '{"type":"control_request","request_id":"x","request":{"subtype":"list_models"}}' \
  | claude --print --verbose --input-format stream-json --output-format stream-json --strict-mcp-config
```

On `claude` 2.1.90 that answers `Unsupported control request subtype: list_models` in about two
seconds and exits 0. [Multica](../Reference/multica-main) uses it in production against 2.1.223 and
2.1.258 (`server/pkg/agent/claude_models.go`), which is where this came from. **Build the adapter to
try the control request and fall back to a static catalogue** — that is Multica's shape, it needs no
version gate because an old CLI answers rather than hangs, and it upgrades itself the day the owner
updates his CLI.

`codex` has no equivalent: an invalid model returns a 400 naming no alternatives, and the valid set
is account-dependent (`"not supported when using Codex with a ChatGPT account"`).

- **If wrong:** a model offered in the picker fails at invocation time with a provider-side error.
  Recoverable and obvious, but it lands on the owner mid-conversation rather than at startup.
- **Clears when:** the daemon asks `claude` and `agy` at runtime and treats an unknown-model error
  as a reason to refresh, with a static catalogue only as the fallback. `codex` cannot be closed
  this way and stays curated.

## G-9 — `claude` cannot authenticate from a spawned process

**Kind:** caveat
**Raised:** 2026-09-10, first end-to-end turn through the daemon.

The daemon runs `claude` as a plain child process. It reads
`~/.claude/.credentials.json`, finds an expired OAuth access token, and has no
way to refresh it — that is the desktop app's job. Ten retries of
`401 authentication_failed`, then:

> `"OAuth access token has expired. Re-authenticate to continue."`

**The failure disguises itself.** The final `result` event carries
`"subtype":"success"` with `"is_error":true`. Reading only the subtype — which
is what I did on the first pass, and reported to the owner as "OAuth works" —
turns a total failure into an apparent success. Anything checking claude's
outcome must read `is_error`.

`codex` and `agy` authenticate fine from the same spawned context, so this is
specific to claude.

- **If wrong:** nothing. It is reproduced on demand and blocks one provider of
  three; the other two work end to end.
- **Clears when:** the owner runs `claude setup-token` and sets
  `CLAUDE_CODE_OAUTH_TOKEN` — [`runbooks/claude-headless-auth.md`](runbooks/claude-headless-auth.md).
  The adapter already exempts that variable from env scrubbing, so the fix needs
  no code change. Confirm with a real claude turn whose `is_error` is false.

## G-10 — Provider headroom is never populated

**Kind:** unproved
**Raised:** 2026-09-10, wiring the surface to real data.

`Provider.Headroom` is always nil, so the strip shows "no limit data" for all
three providers. For `codex` and `agy` that is the truth. **For `claude` it is
not** — it emits a verified `rate_limit_event` with percent and reset time, and
the adapter simply does not parse it yet.

The design's whole honesty argument was that "no limit data" must be
distinguishable from "plenty left". That still holds, but claude is currently
being shown as unknowable when it is merely unparsed, which is its own kind of
lie.

- **If wrong:** the owner cannot see his claude headroom, which is the one place
  it exists — and US3 of the spec is about exactly that.
- **Clears when:** the claude adapter reads `rate_limit_event` and populates
  Headroom, and the strip shows a real percentage against a real reset time.

## G-11 — Server state is held in `useState`, not TanStack Query

**Kind:** caveat
**Raised:** 2026-09-10, wiring the surface to real data.

`AGENTS.md` §3 makes this a hard constraint: TanStack Query owns anything from
the server, Zustand owns view state, and realtime events invalidate or patch the
Query cache. The surface currently holds conversations and the open transcript
in `useState` and patches them by hand from websocket events. **Neither library
is installed.**

It works, and the hand-patching is deliberately written to be the shape a Query
cache update would take. But the constraint exists to prevent a bug class —
stale reads, two sources of truth for one fact, refetch-on-focus done ad hoc —
and this code is inside that class rather than outside it.

- **If wrong:** the failures are the quiet kind. Two tabs disagree; a conversation
  edited elsewhere shows a stale title until reload; an event that arrives during
  a fetch loses the race and is silently overwritten.
- **Clears when:** TanStack Query owns the server state, websocket events patch
  its cache, and `useState` in `chat-surface.tsx` holds only draft, pending
  switch and selection.

## G-12 — The Go server and daemon have no tests

**Kind:** unproved
**Raised:** 2026-09-10, first backend slice.

Everything was verified by running it — a real conversation moved codex → agy →
codex, with the replay quoted, charged and recorded correctly. That is real
evidence and it is what the owner cares about. But there is not one `_test.go`
file, so none of it is protected against the next change.

Multica's rule is the one to adopt when they are written: **no default test may
execute a real agent CLI.** We drive the same binaries on the same machine, and
a test that resolves `claude` from PATH spends the owner's quota. Fake
executable paths by default; real-agent smoke behind a build tag and an env var.

- **If wrong:** a regression in replay, seq allocation or seen_seq lands silently.
  The seen_seq high-water logic is the sharpest edge — it is the difference
  between replaying two messages and replaying two hundred.
- **Clears when:** `go test ./...` covers the store's seq and seen_seq rules, the
  three stream parsers against captured JSONL fixtures, and the turn lifecycle
  with a fake backend.
