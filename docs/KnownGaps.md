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

## G-1 — What `claude` actually emits in print mode is still unverified

**Kind:** unproved
**Raised:** 2026-09-09, while writing [`Capabilities.md`](Capabilities.md). **Narrowed:** 2026-09-10.

Originally covered all three live providers. `codex` and `agy` are now closed — both were run for
real, their streams captured, and `Capabilities.md`'s table reflects actual field names, not
assumed ones.

**`claude` remains open**, and for a specific reason: capturing it from this coding session failed.
`claude -p` hung and returned no output (`exit 124` on a 20s timeout) when run from this session's
Bash tool, and a `--verbose --output-format stream-json` attempt logged repeated
`api_retry` / `authentication_failed` (401) events before being stopped. `codex` and `agy` ran
clean from the identical shell. The most likely explanation is that this session's Bash tool is
itself a nested Claude Code process, and doesn't carry the `claude` CLI's own stored OAuth session
the way a real terminal or the production daemon would — but that is a hypothesis, not confirmed.

- **If wrong** (i.e., if `claude` actually fails the same way from the real daemon): any
  claude-specific field in a chat design is undeliverable, discovered at the backend step instead
  of the design step — the exact waste the design-driven workflow exists to prevent.
- **Clears when:** `claude -p --output-format stream-json --verbose` is run once from a real
  terminal (the owner's, not a nested agent session) or from the actual daemon once it exists, the
  stream is captured, and `Capabilities.md`'s claude column is rewritten as verified.

## G-2 — `codex exec` loads the owner's global MCP config

**Kind:** caveat
**Raised:** 2026-09-10, while capturing `codex`'s real stream for G-1.

A trivial `codex exec --json "Reply with exactly: OK"` produced `AuthRequired` stderr errors for
Supabase and GitHub Copilot MCP servers — both configured in the owner's global `CODEX_HOME`,
neither relevant to this project. Left alone rather than fixed now, because the daemon doesn't
exist yet to configure.

- **If wrong** (i.e., this is fine to ship as-is): every real `codex` run in production logs noise
  for integrations the conversation never asked for, and a misconfigured global MCP server could
  someday do more than log — a tool call landing somewhere unintended.
- **Clears when:** the daemon's `codex` adapter is built with an isolated or minimal `CODEX_HOME`
  (or codex's equivalent of `claude --bare`), verified by a capture showing no unrelated MCP
  activity.

## G-3 — `claude -p` inherits the entire personal Claude Code environment

**Kind:** caveat
**Raised:** 2026-09-10, from the same capture attempt as G-1.

The one `claude -p` invocation that got far enough to emit output (before hanging on auth) showed
a `system.init` payload listing 60+ personal skills, 2 MCP servers, and this machine's full plugin
set — none of it related to the chat feature. Confirms `claude --bare` (or equivalent scoping) is
required, not optional, for the daemon's adapter.

- **If wrong:** every claude-driven chat turn in production silently has access to unrelated
  skills and tools, and starts slower than necessary loading them.
- **Clears when:** the daemon's `claude` adapter passes `--bare` (or the scoping it implies) and a
  capture confirms a minimal `system.init` payload.
