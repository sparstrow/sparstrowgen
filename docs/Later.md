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

## L-2 — Managed daemon installer and supervisor

**Status:** parked **Raised:** 2026-09-09
**Selected next:** 2026-09-12; its trigger is met.

`apps/desktop/` is an empty placeholder. If a desktop shell is needed, its real job is being the
daemon's installer, supervisor, and updater — not a second UI. The selected outcome is a per-user
Windows installation that starts automatically and updates safely. Whether that needs a Go tray,
a small launcher or the eventual Electron shell is a feasibility decision for the feature, not a
reason to make users run the daemon in a terminal.

**Unblocks when:** someone other than the owner needs to run a daemon, or starting it manually
becomes a daily annoyance.

The trigger is now met. The selected direction is a per-user Windows supervisor that installs,
starts and safely updates the daemon; the feature process still determines its approved contract
and design. See [`runbooks/release-workflow.md`](runbooks/release-workflow.md), phase 1.

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

## L-16 — Pair a machine from the browser, and a screen that lists them

**Status:** parked
**Selected next:** 2026-09-12; its trigger is met.
**Raised:** 2026-09-11, while building accounts against Multica's model

`DAEMON_TOKEN` is one static shared secret, the same on the server and on every machine that dials
in. It cannot be revoked for one machine, it does not expire, it says nothing about WHICH machine
is connected, and rotating it means editing configuration in two places at once.

Multica (`Reference/multica-main`) does the better thing, and their Runtimes screen is the proof of
what it buys: `multica login` opens a browser, the already-signed-in human approves, and the server
issues that machine its own token — a hashed row bound to one daemon id, with an expiry, revocable
on its own. The prefix on the token (`mdt_` for a daemon, `mul_` for a user) lets the middleware
route by credential type and fail closed on anything it does not recognise.

What it would give us: a "Computers" screen listing every machine that has been paired, when it was
last seen, which agent CLIs it found, and a button to revoke one — plus `sparstrowgen login` on a
new machine instead of copying a secret into a service file.

**Trigger:** a second machine, or the first time a machine needs to be revoked without disturbing
the other. Browser pairing and disconnecting one computer are now in the approved
[`first usable release`](specs/2026-09-12-first-usable-release.md). What stays parked here is the
rest of the screen: listing every computer with last-seen, found agent CLIs, renaming and model
inventory.

## L-17 — Isolated pull-request preview deployments

**Status:** idea **Raised:** 2026-09-11, configuring the first Coolify deployment

When a pull request is opened, Coolify can build a temporary copy of the application and put its
status and URL on the pull request. From the owner's side, that means opening the proposed version
in a browser and verifying it before merging rather than learning what it does from production.

Do not enable it against the current production configuration. A preview needs its own disposable
Postgres database and its own matching web/API origin pair. Reusing production's `DATABASE_URL`
would let a proposed migration change real data before the pull request is approved; reusing
`WEB_ORIGIN` or `API_ORIGIN` would make the temporary domains fail CORS, cookies, or websockets.
Coolify's GitHub App therefore does not get pull-request write access until the isolation exists.
Normal pushes to `main` can still deploy automatically without that permission.

**Trigger:** a preview can be provisioned with an isolated database, unique web and API domains,
matching origins, and automatic cleanup when its pull request closes.

## L-18 — Administration: invite people, approve or decline access requests

**Status:** parked **Raised:** 2026-09-12, owner, reviewing the account-access prototype

A separate, protected administration place where the owner invites people by email and approves
or declines requests from people he did not invite. Drafted as
[`specs/2026-09-12-admin-invitations-and-approvals.md`](specs/2026-09-12-admin-invitations-and-approvals.md).
Not needed to reach the phase 2 gate: the first usable release records uninvited sign-ups as
requests, emails the owner, and approval means allowing the address in hosting configuration.

**Unblocks when:** the owner has invited or approved people by editing the hosting configuration
more than twice, or before sparstrowgen is offered to anyone he does not know personally —
whichever comes first. Build it on the phase 2 workflow, not before activation.
