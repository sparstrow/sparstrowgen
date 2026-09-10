# sparstrowgen

One chat window for every coding agent you have installed.

Conversations live in our database, not in any provider's session. When you hit a rate limit
on one provider, switch the conversation to another and keep going — the new provider gets
replayed the history it hasn't seen.

## Stack

| Layer | Choice |
| --- | --- |
| Web | Next.js (App Router) + shadcn/ui |
| Server | Go — HTTP API, WebSocket hub, orchestration |
| Daemon | Go — runs on your machine, spawns agent CLIs next to your code |
| Database | PostgreSQL + pgvector (hosted in Coolify) |
| Deployment | Coolify on a Hostinger VPS |
| Desktop | Electron — later |
| Mobile | Expo / React Native — later |

Agent CLIs are driven, not shipped: `claude`, `codex`, `agy` — available now. `gemini` is
installed but blocked (no account signed in); see [`docs/Capabilities.md`](docs/Capabilities.md).

## How it gets built

Design-driven, and ordered so each step is cheaper than the one after it. A feature starts with a
short spec — what the owner wants and why, in his words. Then two to four directions are
*generated as images* (about two minutes each, via codex) and he picks a direction from pictures.
Only that direction gets built as a clickable prototype, only the confirmed prototype gets wired
into the real app on placeholder data, and only then does the backend get built to serve exactly
that. There is no plan document.

The guardrail is [`docs/Capabilities.md`](docs/Capabilities.md): never design something the
backend can't deliver. Full workflow in [`AGENTS.md` §2](AGENTS.md).

## Layout

```
apps/
  web/          Next.js web app
  desktop/      Electron (later)
  mobile/       Expo (later)
packages/
  core/         Headless — API client, query hooks, stores. No UI, no platform APIs.
  ui/           shadcn primitives only. No business logic.
  views/        Shared business views for web + desktop. No next/*, no router imports.
  protocol/     TypeScript types generated from proto/
  tsconfig/     Shared TypeScript config
  eslint-config/
server/         Go module producing both binaries
  cmd/server/   API server — deployed to Coolify
  cmd/daemon/   Local daemon — runs on your machine
  internal/
    agent/      CLI adapters: claude, codex, agy (gemini blocked — no account)
    api/        HTTP handlers
    hub/        WebSocket hub, daemon connection registry
    store/      Database queries
    tunnel/     Preview reverse tunnel (localhost dev server -> browser)
    protocol/   Go types generated from proto/
  migrations/
proto/          Wire protocol — source of truth for server, daemon, and web
deploy/         Dockerfiles and Coolify configuration
Reference/      Read-only reference checkouts (multica)
```

Server and daemon share one Go module deliberately: the wire protocol structs are used by
both, so drift becomes a compile error instead of a runtime bug.

## Architecture

```
Browser / phone ──WSS──┐
                       ▼
            Coolify: server + Postgres
                       ▲
                       └──WSS (outbound, token)── Daemon on your machine
                                                        │ spawns
                                        claude · codex · agy
```

The daemon dials out only. Nothing inbound to your machine, no ports exposed.
