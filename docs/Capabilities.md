# Capabilities — what the backend can actually deliver

**Read this before designing anything.** Design-driven development only works if the design stays
inside what the backend can produce. A screen showing data no provider emits is waste — it looks
finished, the owner approves it, and then the backend can't serve it.

This file is the feasibility surface. It is not a plan and not a roadmap: it says what is
*possible*, not what is scheduled.

Mark every claim as **verified** (we ran it and saw the output) or **assumed** (looks true from
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

## What the agent CLIs emit

This is the hard ceiling on what a chat UI can show. We drive these CLIs; we don't control what
they report.

| | `claude` 2.1.90 | `codex` 0.153.4 | `agy` 1.1.27 |
|---|---|---|---|
| Non-interactive | `-p` *(verified)* | `codex exec` *(verified)* | `-p` *(verified)* |
| Streaming JSON | `--output-format stream-json` *(verified flag)* | `--json` JSONL *(verified flag)* | `--output-format stream-json` *(verified flag)* |
| Token-level partials | `--include-partial-messages` *(verified flag)* | unknown | unknown |
| Resume a session | `--resume <uuid>` *(verified flag)* | `codex exec resume <id>` *(verified flag)* | `--conversation <id>` *(verified flag)* |
| We choose the session id | `--session-id <uuid>` *(verified flag)* | no | no |
| Model override | `--model` *(verified)* | `-m` *(verified)* | `--model` *(verified)* |
| List available models | no — curated list | no — read config | `agy models` *(verified, returns id + label)* |

`gemini` 0.49.0 is installed; its surface has not been checked. Treat it as unknown.

**The field-by-field content of each stream has not been verified.** We know the flags exist; we
have not yet captured and compared actual event streams. Until that happens, do not design any
surface that depends on a specific field being present in all three.

## Do not design these yet

Each of these looks reasonable and is not currently deliverable. If a design needs one, say so
before building it, not after.

- **A uniform per-message token count or cost figure.** Providers do not report usage the same way,
  and some may not report it per message at all. A cost display that is accurate for one provider
  and blank for another is worse than none.
- **A live model picker populated for every provider.** Only `agy` lists its models. For the others
  the list is curated by us and will drift from what the account can actually use.
- **Anything assuming a shared session across providers.** No CLI can resume another's session.
  Switching provider means replaying history into a fresh session — so a design implying one
  continuous thread with the provider is a lie the backend cannot make true. The *conversation* is
  continuous; the provider session is not.
- **Instant response to a keystroke that requires the machine.** The round trip is too long.
- **Anything requiring the agent to control the desktop** — mouse, keyboard, screen pixels. Not
  built, and deliberately not planned. See [`Later.md`](Later.md) L-1.

## Safe to design against

- **Streaming assistant text, token by token.** This is the core interaction and it works.
- **A persistent conversation that outlives any single provider session**, because our database is
  the transcript of record.
- **Switching provider mid-conversation**, with the new provider replayed the history it hasn't
  seen.
- **Per-run metadata**: which provider, which model, when it started and ended, whether it
  succeeded or failed.
- **Provider and model discovery from the machine** — the daemon can probe what's installed and
  report it.
- **Daemon online/offline state**, and which directories are registered.

## Keeping this honest

When a design asks for something not listed here, there are three legitimate answers, and
"probably fine" is not one of them:

1. **Check it.** Run the CLI, capture the stream, and add a verified row. Cheapest option, and
   usually minutes.
2. **Design around it.** Change the design so it needs only what's deliverable.
3. **Record it as a gap.** If the design genuinely needs it and it isn't deliverable yet, that goes
   to [`Later.md`](Later.md) with a trigger, and the design ships without that piece.
