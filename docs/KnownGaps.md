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

## G-3 — `claude -p` inherits the entire personal Claude Code environment, with working tool access

**Kind:** caveat
**Raised:** 2026-09-09, sandboxed capture attempt. **Escalated:** 2026-09-09, real-terminal capture.

Two captures, same finding, worse the second time. The sandboxed attempt showed a `system.init`
payload listing 60+ personal skills and MCP servers in `"pending"` status. The real-terminal
capture — a genuine account, a genuine turn — showed `clockify`, `square`, and `shadcn` all
`"status":"connected"`. **Not merely listed: reachable.** A conversation whose only prompt was
"reply with exactly: OK" had working access to the owner's time-tracking and invoicing tools for
the length of that turn.

This is not a performance or noise concern. It is a scope leak: an unscoped `claude -p` call gives
the model real tool access unrelated to the conversation it's actually in, and the model deciding
not to use it this time is not a boundary — it's luck.

- **If wrong:** every claude-driven chat turn in production has functional access to whatever MCP
  servers happen to be configured on the machine, not just visibility into them. A future prompt
  or an unexpected model decision could act on that access.
- **Clears when:** the daemon's `claude` adapter passes `--bare` (or equivalent scoping) and a
  capture confirms both a minimal `system.init` payload and zero connected MCP servers.

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

Cannot be captured from a sandboxed agent shell; `claude -p` needs the owner's real terminal — this
is what G-1 established.

- **If wrong:** claude renders per-message like codex instead of per-token. A polish downgrade, not
  a broken feature — and the design already has to tolerate a non-streaming provider because of
  codex, so nothing designed against this assumption gets thrown away.
- **Clears when:** one capture of `claude -p --output-format stream-json --include-partial-messages
  --verbose` from a real terminal, checked for `stream_event` / `content_block_delta` entries.
