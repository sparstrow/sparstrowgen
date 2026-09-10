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

## G-13 — Syntax highlighting covers only lowlight's `common` set

**Kind:** caveat
**Raised:** 2026-09-10, while adding the language header to code blocks (B-4)

`rehype-highlight` registers lowlight's `common` languages by default — roughly 37, including go,
python, typescript, sql, bash, java, rust, json, yaml, xml. Not included: **powershell**,
dockerfile, toml, protobuf. A block in one of those is labelled correctly and rendered as plain
text; rehype-highlight emits a `missing-language` message and moves on, so nothing is thrown and
nothing is lost.

Verified: a ```` ```powershell ```` block renders with the header "PowerShell" and zero `hljs-*`
spans (`apps/web/components/chat/markdown.tsx`).

Left alone deliberately. Registering more languages needs `highlight.js` as a direct dependency —
lowlight only re-exports it, and importing through it would be a phantom dependency — and the choice
of which to add is a question about what the owner actually writes in, not one to answer by guessing.
PowerShell is the likely first, on a Windows machine whose own commands are PowerShell.

- **If wrong:** a PowerShell or Dockerfile answer is monochrome. Cosmetic; the code is complete,
  correct, copyable and correctly labelled.
- **Clears when:** the owner says which languages matter, and `highlight.js` is added with just
  those registered.
