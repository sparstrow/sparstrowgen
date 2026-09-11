# Later

Everything not being built right now, in one place, so parking something is never a decision about
which file it goes in.

```
## L-n — <the thing>
**Status:** question | parked | idea      **Raised:** <YYYY-MM-DD>
<Two or three lines. What it is, and why it isn't happening now.>
**Unblocks when:** <a concrete trigger — an event, a threshold, a person. Not "later".>
```

| Status | Meaning |
|---|---|
| `question` | Needs a decision from the owner. Blocks only the one thing that depends on it — build everything else and report "done except L-n" |
| `parked` | Agreed in principle, deliberately not now |
| `idea` | Noticed, no commitment, may never happen |

Write it in the same turn the owner says "park it" / "later" / "just an idea". Delete an entry when
it's done or dead — the length of this file is a real signal. Ids are never reused.

**A `question` should be rare.** The standing preference is that agents recommend and the owner
vetoes. And if a choice can be *rendered*, show him designs instead of writing it down here.

---

## L-1 — Remote desktop access to the machine

**Status:** idea **Raised:** 2026-09-09

Seeing and controlling the machine from a phone. **Not built here** — commodity, multi-year work;
deploy MeshCentral or Guacamole and embed the viewer behind our own auth. Reasoning in
[`Decisions.md`](Decisions.md) D-006.

Do not confuse it with the **preview tunnel** (seeing the app the agent just built on localhost),
which is a planned feature and strictly better for that job — crisp text instead of video, real
touch interaction, no encoding latency.

**Unblocks when:** something needs reaching a *native* app on the machine, not a web app.

## L-2 — Desktop app (Electron)

**Status:** parked **Raised:** 2026-09-09

`apps/desktop/` is an empty placeholder. The desktop app's real job is being the daemon's
installer, supervisor, and updater — not a second UI. Until then the daemon runs as a terminal
process, then a Windows service. A Go system-tray binary should be evaluated before Electron,
since the daemon is already a Go binary and the web UI already exists.

**Unblocks when:** someone other than the owner needs to run a daemon, or starting it manually
becomes a daily annoyance.

## L-3 — Mobile app (Expo)

**Status:** parked **Raised:** 2026-09-09

`apps/mobile/` is an empty placeholder. The web app is responsive and the preview tunnel is
designed for a phone browser, so a phone is reachable before any native app exists.

**Unblocks when:** the owner wants push notification when a run finishes or needs a decision —
which a web app can't do well.

## L-4 — Permission approval UI

**Status:** parked **Raised:** 2026-09-09

Agents run auto-approved inside directories registered with the daemon, which enforces the
boundary rather than trusting the CLI. The approve/deny round-trip is **additive** — a second mode
alongside auto, not a replacement — which is why building auto first isn't throwaway work. See
[`Decisions.md`](Decisions.md) D-007.

**Unblocks when:** an agent must run against a directory the owner doesn't fully trust, or a second
person uses the app.

## L-5 — Skills to write when the thing they describe exists

**Status:** parked **Raised:** 2026-09-09

Carried over from the previous attempt but not imported, because their mechanics described that
codebase. Originals at `D:\My Setup\.claude\skills\` for reference.

| Skill | Write it when |
|---|---|
| Frontend wiring | `apps/web` is scaffolded and has real wiring to describe |
| Go↔TS contract | The first `proto/` message is written |
| Data modelling | The first migration is written — sqlc and pgvector, no row-level security |
| Release | Deploying to Coolify |
| Worktree orchestration | More than one agent runs on a feature at once |

Multica's multi-worktree environment registry is the best version of that last one and worth
copying when it's needed — see [`Decisions.md`](Decisions.md) D-009.

**Unblocks when:** the row's condition is met. Not before — writing them earlier means documenting
an app that isn't built.

## L-7 — Extract shadcn primitives to `packages/ui`

**Status:** parked **Raised:** 2026-09-10

`apps/web/components/ui` holds the shadcn primitives, not `packages/ui` as the locked layout says.
With one consumer, a package boundary would be a speculative abstraction (AGENTS.md rule 4) and
would add workspace wiring before anything works. The move is mechanical when it's earned.

**Unblocks when:** a second consumer exists — `apps/desktop` (L-2) or `packages/views` gaining a
view that the web app and something else both render.

## L-9 — Name conversations automatically

**Status:** idea **Raised:** 2026-09-10

Every conversation is "Untitled conversation" until renamed by hand, so the sidebar is a column of
identical rows distinguished only by folder and age. Search partly rescues this — it looks inside
message bodies precisely because titles are unreliable — but a list you cannot scan is still a list
you cannot scan.

The obvious approach costs a model call per conversation. The cheap one is the first line of the
first message, truncated, which is free and right most of the time.

**Unblocks when:** the sidebar holds enough conversations that finding one by eye stops working.

## L-10 — The raw view shows the stored text, not the provider's event stream

**Status:** idea **Raised:** 2026-09-10

The Raw toggle (`components/chat/raw-transcript.tsx`) prints the transcript exactly as it is stored,
which is what the markdown renderer is handed — so any difference between the two views is the
renderer's doing, and that is the question it was built to answer.

It cannot answer the question one layer down. A CLI emits far more than its answer: reasoning
events, tool calls, file edits, per-turn metadata. The daemon parses those, keeps the text and the
usage, and discards the rest — so if a provider *said* something we never stored, no view in the app
can show it. Seeing that needs the daemon to retain the raw event stream per turn, which is a
storage decision (how much, for how long) rather than a UI one.

**Unblocks when:** an answer looks wrong in a way the stored text cannot explain — most likely a
turn that used tools, where what the agent *did* is invisible and only what it said survives.

## L-11 — The transcript does not record that a conversation moved

**Status:** idea **Raised:** 2026-09-10, building the folder picker (B-3)

A provider switch is written into the transcript as a marker, at the moment the catch-up is paid
for. A folder move is not, even though it changes the thing the answers are *about* — read back
later, every answer above the move looks like it was about the folder shown in the header now.

The picker says so at the time ("they were about the old folder, and nothing above will say so"),
which is honest but only helps the person who moved it, and only right then.

The cheap version is another marker role alongside `replay`, which the CHECK constraint on
`entries.role` would have to allow. The more accurate version records the folder on each agent
entry, so a transcript can show where each answer actually came from — a column rather than a row,
and no new role.

**Unblocks when:** a conversation is moved and the transcript above it is later misread, or the
folder becomes something that changes often enough to be worth the record.

## L-12 — postMessage has no test

**Status:** idea **Raised:** 2026-09-10, after B-7

B-7 lived in `postMessage`, which has no test at all — `api_test.go` covers `estimateTokens` and
nothing else, because exercising the handler needs a database, a hub, and something pretending to
be a daemon. The store half of the same bug *is* covered
(`TestSetFolderDropsProviderSessions` asserts the whole transcript comes back unseen); the half that
decides whether to actually send a replay was verified only in a browser.

**Unblocks when:** a second bug lands in that handler, or the daemon gets a test double for
something else and the harness is already paid for.

## L-13 — Retry a message

**Status:** idea **Raised:** 2026-09-11, owner

Send the same message again and take the new answer instead. Two situations want it and they are
not quite the same: a turn that **failed** (retry is the obvious repair, and the prompt is already
known), and a turn that **succeeded badly** — the answer was wrong, or went off in a direction that
was not asked for. The second is the common one on a coding agent, and it is also the one that
raises the question of what happens to the reply being replaced.

Interacts with [[L-8]]: a stopped turn is the third case, and it is the one where retry is most
obviously the next thing you want.

**Unblocks when:** the owner re-types the same message a second time to get a different answer.

## L-14 — Rewind a conversation to an earlier point

**Status:** idea **Raised:** 2026-09-11, owner

Go back to how the conversation stood some messages ago and carry on from there, discarding what
came after — Claude Code's `/rewind`.

**One half of it we cannot do, and should not imply.** Claude Code rewinds the *files* as well as
the transcript, because it made the edits and knows what they were. We drive the CLIs as black
boxes: the daemon sees text, usage and a session id, and nothing about what an agent wrote to disk.
So ours would rewind the conversation and leave the working tree exactly as the agent left it —
which is a genuinely useful thing, but it is not what a Claude Code user means by the word, and the
wording has to be honest about that or it will be trusted for something it cannot do.

**Unblocks when:** a conversation goes far enough wrong that starting a new one is the only way out.

## L-15 — Fork a conversation into a new one

**Status:** idea **Raised:** 2026-09-11, owner

Branch off at any point into a separate conversation, leaving the original untouched — so two
approaches can be tried from the same starting context without either destroying the other.

The gentlest of the three, because nothing is destroyed: a fork is a copy of the entries up to a
point, and the copy needs no provider session (the new conversation simply has an unseen
transcript, which the catch-up already handles as a solved problem).

**All three of these hit the same wall.** `entries.seq` is dense and gapless per conversation
*by design* — `provider_sessions.seen_seq` counts against it, which is what makes a catch-up replay
exactly the gap and nothing more. Retry, rewind and fork all want to remove or diverge from part of
a transcript, and none of them can do that by deleting rows without deciding what a `seen_seq`
pointing past the cut now means. Whichever of the three is built first pays for that decision, and
the other two get it free. Worth doing them as one piece of work rather than three.

