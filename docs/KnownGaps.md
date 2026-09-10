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

## G-1 — What the agent CLIs actually emit is unverified

**Kind:** unproved
**Raised:** 2026-09-09, while writing [`Capabilities.md`](Capabilities.md).

The CLI *flags* are verified — `--output-format stream-json`, `--json`, `--resume`, `--model` and
friends were confirmed by running `--help` on `claude` 2.1.90, `codex` 0.153.4 and `agy` 1.1.27.
**The contents of those streams were not.** No event stream has been captured or compared across
providers, so we do not know which fields exist, which are shared, or what a turn's lifecycle
looks like in each.

`gemini` 0.49.0 is installed and its surface has not been looked at at all.

- **If wrong:** any chat surface designed against an assumed field is undeliverable, and we find
  out at the backend step instead of the design step — the exact waste the design-driven workflow
  exists to prevent.
- **Clears when:** each CLI is run once in print mode, its stream captured, the three compared,
  and `Capabilities.md`'s assumed rows are rewritten as verified ones.
