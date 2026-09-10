# Decisions

Load-bearing technical choices, with what was rejected and why. Append-only; a few lines each, not
a document. The code shows what was built — this is the only place that shows why the alternatives
lost.

Add an entry when a choice would be expensive to reverse: schema, protocol, transport, a boundary
between components. Skip it for choices that are cheap to change later.

`.sparstrowgen/blueprint.yaml` holds *what* the stack is. This holds *why*.

---

## D-001 — Go for both the server and the daemon

**2026-09-09.** Rejected: TypeScript everywhere, Rust everywhere, Go server with a Rust daemon.

The daemon is a genuinely native program — process groups, signals, PTYs, file watching — and
cross-compiles to a single dependency-free binary, which Node cannot match without native addons.
Rust would win on raw capability but its async complexity buys nothing for what is ~90% I/O
orchestration, and compile times matter when iterating with agents.

The one thing that argued for Rust was screen capture and video encoding. That disappeared when
"screen sharing" turned out to mean a reverse HTTP tunnel to a localhost dev server, not pixels.

## D-002 — Server and daemon share one Go module

**2026-09-09.** Rejected: separate modules with a shared protocol package.

Both binaries build from `server/`. The wire protocol structs are used by both, so drift becomes a
compile error rather than a runtime bug on a laptop running an older daemon.

## D-003 — Protobuf for the wire protocol

**2026-09-09.** Rejected: hand-maintained JSON types on both sides.

One `.proto` generates Go structs and TypeScript types. The alternative is the fast option that
gets replaced: the moment an installed daemon lags the server, untyped JSON becomes a runtime bug
instead of a compile error. The codegen step is paid once, in week one.

## D-004 — Our database is the transcript of record; provider sessions are a cache

**2026-09-09.** Rejected: treating a provider's own session as the source of truth.

No agent CLI can resume another's session, so cross-provider switching — the product's whole
reason to exist — is impossible unless we own the conversation in a provider-neutral format. A
`provider_session` row per (conversation, provider) with a last-seen sequence number means
switching back later replays only the gap, not the whole history.

The schema is multi-provider from the first migration for the same reason. Retrofitting it is
exactly the rewrite this project is trying to avoid.

## D-005 — Embeddings run behind HTTP, never in-process

**2026-09-09.** Rejected: an in-process embedding library; a Python service for the whole backend.

pgvector lives in the Postgres we already run; embeddings come from a separate container (Ollama
or text-embeddings-inference). This keeps the backend language decision free of the ML ecosystem
— vector search is SQL, and embedding is an HTTP call. Python only earns a place if we start
training models, and that arrives as its own service behind the same boundary.

## D-006 — Remote desktop is deployed, not built

**2026-09-09.** Rejected: building screen capture, encoding, and input injection.

Commodity, multi-engineer-year work. Deploy MeshCentral or Guacamole and embed the viewer behind
our own auth. Full reasoning in [`Ideas.md`](Ideas.md) I-1.

## D-007 — Agents run auto-approved inside registered directories

**2026-09-09.** Rejected: plan/read-only first; a full permission UI first.

Auto-approval scoped to directories registered with the daemon is what actually replaces the
owner's desktop apps, and the daemon enforces the boundary rather than trusting the CLI. The
approve/deny UI is **additive** — a second mode alongside auto — which is why building auto first
is not throwaway work. Parked as [`Deferred.md`](Deferred.md) D-3.

## D-008 — Design-driven development, no spec or plan documents

**2026-09-09.** Rejected: spec → plan → code, which is what the previous attempt did.

The design is the specification. Features start with two or three rendered directions, the chosen
one gets wired into the real app on placeholder data, and the backend is built only once the owner
has confirmed it. The guardrail is [`Capabilities.md`](Capabilities.md): design nothing the backend
cannot deliver.

The previous attempt spent its budget on planning documents and never shipped. Written specs and
plans, their templates, and their skills were removed rather than kept "just in case" — a
process that exists is a process that gets followed.
