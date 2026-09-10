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

## G-3 — `claude -p` inherits the entire personal Claude Code environment, with working tool access

**Kind:** caveat
**Raised:** 2026-09-10, sandboxed capture attempt. **Escalated:** 2026-09-10, real-terminal capture.

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

## G-4 — Two narrower streaming questions left after the real `claude` capture

**Kind:** unproved
**Raised:** 2026-09-10, after closing G-1.

The real capture answered the shape of a full turn, but not everything:

1. **What does a hit rate limit actually look like?** The captured `rate_limit_event` showed only
   `"status":"allowed"`. We do not know the blocked value, or whether the payload changes shape
   when a limit is actually exceeded. Cannot be forced in a trivial test.
2. **Token-by-token deltas are unverified for all three providers.** Every capture so far used the
   non-partial stream mode — one complete message per turn. The incremental "typing" feel a chat
   UI wants requires `claude --include-partial-messages` (and codex/agy's equivalents, if they
   exist) captured separately.

Lower stakes than G-1 was: (1) blocks the "you're over your limit" state specifically, not the
whole feature, and a design can ship the "approaching your limit" state without it; (2) is a
rendering smoothness question, not a data-modeling one.

- **If wrong:** (1) a "provider blocked" UI state gets built against a guess and needs correcting
  once a real limit is hit. (2) chat text renders per-message rather than per-token until fixed —
  a downgrade in polish, not a broken feature.
- **Clears when:** (1) any provider is used enough to actually hit a limit, or provider
  documentation describes the blocked payload. (2) one more capture with
  `--include-partial-messages` added.
