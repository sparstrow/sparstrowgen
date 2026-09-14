# Bugs

Wrong behaviour in the running app — owner-reported or agent-found. Logged in the same turn it
surfaces, because a problem mentioned only in chat does not exist to the next session.

Entries are marked resolved in place, never deleted, so the record survives. Ids are never reused.

Format — keep it to this, no template needed:

```
## B-n — <what is wrong, in one line>
**Found:** <YYYY-MM-DD>, <by whom / during what>   **Status:** open | fixed <date>
**Repro:** <the shortest reliable path to see it>
**Expected / Actual:** <one line each>
**Security:** <only if it is a trust-boundary issue — auth bypass, data crossing users,
a leaked credential. Never paste a live secret or a working exploit payload.>
```

---

## B-1 — A long model menu ran off the bottom of the screen, hiding items

**Found:** 2026-09-10, verifying the two-dropdown switcher in a browser **Status:** fixed 2026-09-10
**Repro:** Pick `agy` in the composer's provider dropdown, open the model dropdown. It has 14 entries.
**Expected / Actual:** The whole menu is reachable / it rendered 320px tall from y=605 in a 734px
viewport — bottom edge at 909, so roughly 175px of it sat below the fold with no way to scroll to it.

Cause was mine and worth remembering: `DropdownMenuContent` already caps itself at
`max-h-(--available-height)`, which is what lets Base UI's positioner fit a popup into the space
that actually exists. Passing `className="max-h-80"` overrode that with a fixed height, so the
popup stopped adapting. Fixed by dropping the override and passing `side="top"` on both composer
menus, which are anchored at the bottom of the window. Re-verified at 734px (fits, no scroll needed),
500px (fits), and 360px (caps at 294px and scrolls).

**The general rule:** never set a max-height on a popup. The library is already doing it, and
overriding it converts "fits the screen" into "fixed size, may not fit".

## B-2 — Agent answers render as plain text; code blocks and markdown are lost

**Found:** 2026-09-10, reviewing what is left to build **Status:** fixed 2026-09-10
**Repro:** Ask any provider for code. The reply arrives as one unbroken paragraph.
**Expected / Actual:** Fenced code renders as a code block, as the locked design shows and as
`AgentMessage.code` in the types provides for / everything renders as plain text, fences and all.

The design that was approved had a syntax-highlighted code block in it, and `code?: {lang, body}`
exists in `chat-types.ts` for exactly that. Nothing ever populates it: the server stores the reply
as one string and `message-list.tsx` prints it with `whitespace-pre-wrap`.

This is the single biggest functional gap for a **coding**-agent chat — most useful answers are
mostly markdown, and headings, lists, inline code and fenced blocks are all currently flattened.

**Fixed** by rendering the body as markdown (`components/chat/markdown.tsx`: react-markdown +
remark-gfm + rehype-highlight) rather than extracting a single `code` field. That field was a
mock-era artefact of a design drawn around one example answer — real replies interleave prose with
several code blocks, which one field cannot represent — so it is deleted from the types.

Syntax colours are our own tokens (`--code-keyword` and five siblings) mapped onto the `hljs-*`
classes, not an imported highlight.js theme: every one of those ships literal hex and would be
correct in exactly one of our two themes. Code blocks scroll inside their own box and carry a copy
button.

Verified against a real codex answer containing a heading, a list, a fenced Go block and a table:
all rendered, six syntax tokens highlighted, no raw fences left on screen.

## B-3 — Every conversation runs in the server's working directory

**Found:** 2026-09-10, reviewing what is left to build **Status:** fixed 2026-09-10
**Repro:** Create a conversation. Its folder is whatever directory the server process was started
in, and there is no way to change it.
**Expected / Actual:** The owner picks the project a conversation is about (spec US1: "in a
conversation tied to the project I'm working on") / every conversation claims to be about
`D:\sparstrowgen` regardless.

The API already accepts a folder on create and the daemon already runs the CLI in it — `POST
/api/conversations` takes `{folder}` and it is honoured. Nothing in the UI sends one, and there is
no way to change it afterwards.

**Why it matters more than it looks:** the folder is what the agent can see. A conversation about
the clinic project that silently runs against this repo gives confidently wrong answers about the
wrong codebase, with nothing on screen saying so.

**Fixed 2026-09-10.** The folder shown in the header is now the button that changes it, opening a
picker with three ways in: recent folders (derived from conversations that already exist, so there
is no second list to maintain), a browsable tree, and a paste-a-path field with a visible **Go**
beside it. New conversations inherit the folder last used rather than the server's working
directory, which alone fixes the common case.

Browsing is answered by the daemon, never by the server — see D-020 for why that is not
over-engineering. Paths are stored as the daemon resolved them, so `..` and a typed-in spelling
never reach the database.

**Two things surfaced while verifying it, both of which only appear once you actually move a
conversation and run a turn:**

- claude keys its sessions by project directory. A resume id recorded in the old folder answers
  `No conversation found with session ID` from the new one, and every later turn failed in about a
  second. `SetFolder` now drops the conversation's provider sessions, which is right for all three
  CLIs regardless: a session built elsewhere is reasoning about the wrong tree.
- Dropping them exposed B-7 below.

## B-7 — A caught-up provider only replayed on a provider switch

**Found:** 2026-09-10, verifying the B-3 fix **Status:** fixed 2026-09-10
**Repro:** Move a conversation to another folder, then ask a question about something said earlier.
**Expected / Actual:** the agent has the conversation / it starts blind, with no history and nothing
on screen saying so.

`postMessage` gated the catch-up on `body.Provider != conv.Provider` — which quietly assumed a
provider switch is the only way a session goes missing. Moving a folder drops the sessions too, so
the next turn ran with `replay=0` against a brand-new session.

The condition is now simply "this provider has not seen these entries", which is what `seen_seq`
already answers and covers every reason a session disappears, including ones not invented yet.

The marker's wording had the same assumption baked in: it said *switched to claude* when nothing
had switched. `MessageList` now decides that by looking at which provider actually answered last,
so a same-provider catch-up reads **caught up claude · replayed 8 messages**.

Verified end to end: after moving a conversation into `docs/`, one question got both halves right —
the new working directory, and the first thing asked in the conversation, which only the replay
could have supplied.


## B-4 — Fenced code blocks threw away the language and everything else on the fence line

**Found:** 2026-09-10, owner, reading the first markdown-rendered answers **Status:** fixed 2026-09-10
**Repro:** Ask any provider for code in several languages. Every block renders identically, with
nothing saying which language it is.
**Expected / Actual:** A block declared ```` ```go ```` is labelled Go, and anything else on that
fence line (```` ```go title="main.go" ````) is shown / both were parsed and then dropped.

B-2 rendered the block but not what the agent said *about* the block. The information was never
missing from the reply — remark parses the fence's language into a `language-go` class and its meta
string into `data.meta`, and the renderer read neither.

**Fixed** in `components/chat/markdown.tsx`: a header bar on each block carrying the language, the
meta string when there is one, and the copy button (previously floating, and revealed only on
hover).

**The part worth not undoing:** the language is captured by a remark plugin *before*
rehype-highlight runs. With `detect` on, the highlighter guesses a language for untagged fences and
records the guess as a class of exactly the same shape — after it has run, the agent's word and the
machine's guess are indistinguishable. Verified on a real untagged SQL block: it was guessed as CSS.
A header presenting that as the language would be worse than no header, so an untagged fence gets no
label and keeps its (still useful) highlighting.

Verified in the browser against four fences — `go` with a meta string, untagged, `zsh` (no display
name, shown verbatim), `powershell`: labels correct in both themes, each copy button copying its own
block. See G-13 for what stays uncoloured.

## B-5 — codex replies with a preamble render as shredded inline code

**Found:** 2026-09-10, owner, asking codex for a C++ tree **Status:** fixed 2026-09-10
**Repro:** Ask codex `give me code block with dsa tree in c++`. The answer arrives with the prose
running into `` ```cpp ``, then fragments of the program in small disconnected boxes.
**Expected / Actual:** one paragraph followed by one code block / the fence is swallowed and the
program is torn into pieces of inline code.

**Cause, verified against a real capture** (`server/internal/agent/testdata/codex-two-messages.jsonl`):
one codex turn can complete **two** `agent_message` items — a preamble sentence and then the code —
and `parseCodex` concatenated them with no separator:

```
item 1 (78ch)   "I'll use a binary search tree with insertion, search, deletion, and traversal."
item 2 (2432ch) "```cpp\n#include <iostream>\n…"
```

Joined edge-to-edge that is `…traversal.```cpp`, and a fence that does not begin a line is not a
fence at all — CommonMark reads the backticks as an inline code span, which then pairs off with the
next backtick run and shreds the rest of the reply.

Nothing was wrong with what codex sent. The markdown renderer was innocent too; it rendered exactly
what it was given.

**Fixed** in `server/internal/agent/codex.go`: separate messages are joined with a blank line.
Regression test `TestParseCodexSeparatesMessages` asserts no line contains a fence that does not
start it, and was confirmed to fail on the old code with the exact welded line.

**Found by the Raw view** added the same day — the rendered pane looked like a rendering bug, and
the raw pane showed the fence sitting mid-line in the stored text, which put the search in the
backend immediately. Worth remembering the next time something looks like a renderer fault.

**Not repaired retroactively:** transcripts already stored keep the welded text and still render
badly. Rewriting stored replies is a destructive edit to the record and needs the owner to ask.

## B-6 — claude keeps only its last message; everything before a tool call is dropped

**Found:** 2026-09-10, clearing G-14 **Status:** fixed 2026-09-10
**Repro:** Ask claude something that makes it use a tool and speak either side of it — "First write
one sentence saying which file you are about to open. Then read package.json. Then give me a fenced
json code block."
**Expected / Actual:** the sentence, then the code block / only the code block. The sentence is gone
and nothing on screen suggests anything is missing.

**Cause, verified against a real capture**
(`server/internal/agent/testdata/claude-two-messages.jsonl`): one claude turn is **four** assistant
messages, not one.

```
assistant #1  ["thinking"]
assistant #2  ["text"]      "Opening the root package.json to check the package identity."   (60ch)
assistant #3  ["tool_use"]  Read
assistant #4  ["text"]      "```json\n{ … }\n```\n\nNo `version` field is present…"          (174ch)
```

`parseClaude` treated the `assistant` event as the whole answer and called `full.Reset()`, so #4
erased #2. That reset was correct for a different reason — the deltas of *that same message* had
already been accumulated and must not be counted twice — and collapsing the two jobs into one
buffer is what made it wrong.

**Fixed** in `server/internal/agent/claude.go`: `done` holds completed messages, `streaming`
accumulates deltas for the one still arriving, and an `assistant` event supersedes only its own
message. Messages are joined with a blank line for the same reason as B-5 — glued together, a fence
stops beginning its line and stops being a fence. Deltas left with no `assistant` event behind them
are kept, so a turn that dies mid-answer still shows what arrived.

`TestParseClaudeKeepsEveryMessage` runs the real capture and asserts both messages survive, blank-line
separated, with no fence off line start. Confirmed to fail on the old code, dropping exactly the
60-character sentence.

**Also verified in the same capture:** only `text_delta` carries `.text`. `thinking_delta` and
`input_json_delta` do not, so claude's private reasoning and its tool arguments cannot leak into a
transcript — which the fix had to preserve, since it now keeps more than it used to.

**Worse than B-5 was.** B-5 mangled the formatting of text that was all still on screen; this
deleted whole messages silently. Both were the same underlying mistake — treating several messages
as one string — and the raw view is what made the first one findable.

**Not repaired retroactively:** replies already stored are missing that text for good; it was never
written down. Only what claude sends from now on is kept whole.

## B-8 — A turn in flight when the daemon disconnects never ends

**Found:** 2026-09-11, building the stop button (L-8) **Status:** fixed 2026-09-11
**Repro:** Send a message, then kill the daemon before the reply arrives.
**Expected / Actual:** the turn is reported as broken and the conversation becomes usable again /
the working indicator ticks forever, the composer stays locked, and the agent entry stays an empty
placeholder — including after a refresh, because the placeholder is a real row.

`daemonSocket`'s deferred cleanup calls `hub.ClearDaemon` and nothing else, so every entry in
`api.turns` is orphaned (`server/internal/api/sockets.go:52`). Only a message carrying that turn id
finishes a turn, and the process that would have sent one is gone.

**Predates the stop button and is not caused by it** — this is what "there is no way to call a turn
back" looked like even before there was a button. But it is the same shape of problem: a turn that
cannot end. The fix is small (finish every in-flight turn when the daemon goes, with the failure
saying the machine went away) and deliberately not bundled into L-8, which is already a protocol
change, a migration and a process-tree change.

**The second half, fixed 2026-09-11:** a machine that is present but silent. This was the part left
open, and it is not theoretical — a nested `claude -p` hanging forever is already recorded in D-016,
killed by hand at 120s. The daemon now runs an inactivity watchdog per turn: `TURN_IDLE_TIMEOUT`,
15 minutes by default.

**An inactivity watchdog, not a wall-clock cap**, and the distinction is the whole design. Multica
has a ticket for getting this wrong (MUL-3064): a total timeout kills a session that is working
perfectly well and merely taking a while, which on a coding agent is most real tasks. A turn
streaming for an hour is fine; a turn silent for fifteen minutes is not.

**Why fifteen and not three.** The asymmetry favours patience. A false positive throws away real
work and the quota spent earning it; a false negative just means waiting longer for a net that only
matters when nobody is watching — and since the stop button shipped, somebody who *is* watching ends
a turn in one click. The budget is also tightest on codex, which emits nothing between its session
id and its finished answer, so for codex this is a cap on the whole turn rather than a silence
detector. That is the honest limit of what the CLI tells us.

**Fixed 2026-09-11, on both sides of the socket.**

- **Server:** `abandonTurns` closes out every turn still in flight when the daemon goes, keeping
  whatever text had streamed in, for the same reason a failed turn keeps its partial answer. It
  reads as a failure, not a stop — a laptop closing is not the owner changing their mind.
- **Daemon:** the CLIs are cancelled too. The server has given up on those turns, so anything still
  running is spending the owner's quota on an answer with nowhere to go. Deliberately *not*
  `stop()` in a loop: that would record them as deliberately stopped and put a lie in the
  transcript.

**A prerequisite that was its own latent bug.** `hub.ClearDaemon` guarded the actual clear with
`if h.daemon == c` but broadcast "machine offline" unconditionally — so when a daemon reconnected,
the old socket's late teardown told every browser the machine was unreachable moments after it came
back. Hanging turn-abandonment off that same teardown would have been far worse: it would have
killed the *new* daemon's turns. `ClearDaemon` now reports whether the socket was still current, and
both the broadcast and the abandonment are conditional on it.

Regression tests are the new API harness described under L-12's closure — `TestATurnIsClosedOutWhenTheMachineGoesAway`
confirmed to fail on the old code with exactly the reported symptom (`entry never settled:
text="I had started to say" failure="" stopped=false`). The daemon-identity half is tested in the hub
package instead, because the API-level version passed with the guard removed and a test that cannot
fail is worse than none.

## B-9 — The composer shows the old provider after sending to a new one

**Found:** 2026-09-11, verifying the stop button (L-8) **Status:** fixed 2026-09-11
**Repro:** In a conversation on claude, pick codex in the provider dropdown and send a message.
**Expected / Actual:** the composer says codex, because that is what the conversation is on now /
it snaps back to claude, and the placeholder reads "Message claude…". A reload corrects it.

Purely a stale cache, and the server is right: `postMessage` calls `store.SetProvider` and the
conversation really is on codex (verified directly — `provider: codex`, `model: GPT-5.6 Sol`), but
nothing broadcasts an `EventConversation` for that change, so the browser's copy still says claude
until something else refetches it.

The transcript is unaffected — the working indicator and the agent entry both name codex correctly,
because they take the provider from the send rather than from the cached conversation. It is the
composer alone, which reads `selected.provider`.

**Predates the stop button**; found while verifying it because switching provider and then watching
the composer is not something the earlier rounds happened to do.

**Fixed 2026-09-11.** `store.SetProvider` now returns the updated conversation — the SQL was already
`RETURNING *` and the store was discarding it — and `postMessage` broadcasts it, alongside the
`EventConversation` that `patchConversation` already sends.
`TestSendingToADifferentProviderAnnouncesTheChange` watches a real browser socket and was confirmed
to fail on the old code ("no conversation event arrived").

## B-10 — A failed turn threw away everything a non-streaming provider had said

**Found:** 2026-09-11, watching the daemon log while verifying the idle watchdog **Status:** fixed 2026-09-11
**Repro:** Make a `codex` turn fail or time out after it has produced some output.
**Expected / Actual:** what it had written is kept, as it is for a stopped turn / the entry is
stored empty and the output is gone for good.

**Found by a number, not by the screen.** The UI showed a plausible "Turn did not finish" box and
nothing looked wrong; the daemon log said `chars=425`. The turn had produced 425 characters and
none of them reached the transcript.

`DaemonFailed` carried only an error message, so `handleDaemonMessage` wrote the failure from
`t.Streamed` — the deltas the server had accumulated. That is the right source for `claude` and
`agy`, and **empty for `codex`, which emits no deltas at all**. The text existed the whole time: the
parser had recovered it into `result.Text`, and the daemon simply never sent it.

The same shape as B-5 and B-6 — a provider difference that is invisible until you look at the one
provider that behaves differently — and the third time `codex` not streaming has cost something.

**Fixed** by giving `DaemonFailed` a `Full` field like `DaemonDone` and `DaemonStopped` already
have, and preferring it over the accumulated deltas. Both directions are tested:
`TestAFailedTurnKeepsTextFromAProviderThatDoesNotStream` (confirmed to fail on the old code with
`text = ""`) and `TestAFailedTurnStillFallsBackToTheDeltasItStreamed`, so the fix cannot regress the
streaming case it replaced.

**Predates the watchdog** — any failed codex turn lost its output this way. The watchdog only made
it easy to produce on demand.

**Also fixed in the same change:** the failure box told every broken turn that "what arrived before
it stopped is kept above", including turns where nothing had arrived and there was nothing above.
It now says that only when there is something to mean.


---

## B-11 — `agy` cannot use any tool, so it cannot read the codebase it is pointed at

**Found:** 2026-09-11, checking whether an agent can read a dropped file from disk **Status:** open
**Repro:** In a folder containing `evidence.png`, run the adapter's own command line —
`agy -p "Read the file evidence.png in this directory..." --output-format stream-json`. No flags
beyond those in `server/internal/agent/agy.go`.
**Expected / Actual:** The file is read and described / nothing is produced, and agy says so
plainly:

> `no output produced — a tool required the "command" permission that headless mode cannot prompt
> for, so it was auto-denied. Add an allow-rule under permissions.allow in settings.json (e.g.
> command(<target>)). Alternatively, re-run with --dangerously-skip-permissions to auto-approve
> all tools.`

The other two are not like this. `codex` runs under a `read-only` sandbox by default and read the
same file without any flag from us; `claude` allows its read-only tools in `-p` without prompting.
Only `agy` denies everything it cannot prompt for, and headless mode can never prompt.

So agy in this app can answer from what the model already knows and nothing else — it cannot read a
file, search the tree, or run a command. For a harness whose premise is one chat across every
*coding* agent, that is the provider not doing the job rather than a missing nicety, and it is
invisible today because nobody has asked agy to touch the codebase.

**Not fixed here, because the fix is a decision rather than a flag.** The two routes agy documents
are `--dangerously-skip-permissions` (auto-approve *everything*, which is the opposite of the
scoping this project treats as a security boundary — see D-018) and an allow-rule in the user's own
`settings.json`, which is the owner's tooling config rather than ours to write. `--sandbox` exists
and may be the honest pairing for the first, but the combination is unverified: the check needed to
confirm it was blocked before it ran, and no claim is made here about what it does.

**Blocks a design that assumes every provider can read an attached file** — see
[`Capabilities.md`](Capabilities.md).

## B-12 — Simultaneous sign-in attempts all slipped past the login throttle

**Found:** 2026-09-11, codex reviewing the auth code adversarially before it shipped
**Status:** fixed 2026-09-11

**Repro:** Send a hundred `POST /api/auth/login` requests at the same instant with a wrong
password. All hundred were admitted.

**Expected / Actual:** The allowance is five attempts before delays begin / it was five attempts
*per burst*, because the throttle checked and recorded in two separate lock acquisitions and the
argon2 hash — the slowest thing in the request, ~50ms — sat in the gap between them. Every request
that reached the check before any of them had finished hashing saw an empty counter and was let
through.

**Security:** Yes, two ways. An attacker gets as many guesses per round as they care to send in
parallel, which is most of what a rate limit exists to stop. And each admitted request holds 19 MiB
of argon2 working memory at once, so a hundred of them is about 1.9 GiB — enough to take the VPS
down, which locks the owner out just as effectively as guessing the password would let someone in.

Never shipped: found and fixed in the same change that introduced it. Logged because the shape is
worth remembering rather than because it reached anyone — **check-then-act is not made safe by
locking each half**, and the tell is a slow operation sitting between the check and the record.

Fixed by making admission reserve the attempt in the same lock acquisition that checks it
(`Throttle.Begin`), so a request is counted the moment it is let in rather than after it fails.
Two further hardenings landed with it, both found in the same review: a hash's own parameters are
now bounded before use (a one-byte key would have verified a password on one byte, and `t=0`
panics inside argon2 rather than erroring), and the login body is capped at 4 KiB with at most four
concurrent hashes.

Regression test: `server/internal/auth/race_test.go` — it fails on the old check-then-act shape.

## B-13 — One conversation with an unknown provider white-screened the whole sidebar

**Found:** 2026-09-11, signing in for the first time after auth landed **Status:** fixed 2026-09-11

**Repro:** Have any conversation whose `provider` is not one of claude/codex/agy — creating one
while the daemon is offline leaves it empty — then load the app.

**Expected / Actual:** That row renders as best it can / `Cannot read properties of undefined
(reading 'text')` at `conversation-list.tsx:128`, and the entire app fails to render. Not the row:
the app.

`providerClasses[conversation.provider]` returns `undefined` for anything not in the map, and the
next line reads `.text` off it. Nine call sites across six components did exactly this, so the same
one row would have taken down the composer, the message list, the raw view and the provider strip
just as completely.

This is the case [`AGENTS.md`](../AGENTS.md) §3 names directly — *an installed daemon will one day
be older than the server* — arriving early and by a different route than expected. The fix is one
accessor, `providerStyle()`, that always returns something, plus a neutral mark in `ProviderIcon`
for a provider this build has no logo for. An unknown provider is a build that is behind, not an
error, and it is not coloured as one.

**How the bad rows got there, which is the other half:** the new auth tests created conversations
through `POST /api/conversations` and never deleted them. The harness only cleaned up the ones made
via its own helper. No daemon is connected during a test, so each one was saved with no provider at
all — seven of them, in the development database the app actually runs against. Fixed by giving the
harness a `createConversation()` that registers its own cleanup, and the seven empty rows were
deleted after checking each had zero entries.

## B-14 — Signing out left the app sitting on the signed-out screen's data

**Found:** 2026-09-11, clicking sign out in a browser **Status:** fixed 2026-09-11

**Repro:** Sign in, click Sign out.

**Expected / Actual:** The login form / the request succeeded, the cookie was gone, and the chat
surface stayed on screen showing a conversation the server would now refuse to hand over. A manual
reload then showed the login form, which is what made it clear the server side was fine.

Cause was ordering in `useSignOut`: it called `queryClient.clear()` and then set the session flag.
`clear()` removes the query objects that mounted components are subscribed to, so those observers
are left watching nothing and the `setQueryData` that follows triggers no re-render at all.

Fixed by setting the session flag FIRST — which re-renders the gate, swaps in the login form and
unmounts everything watching a query — and only then removing the rest of the cache.

Worth remembering beyond this bug: `clear()` is not "invalidate everything harder". It detaches
live observers, so anything that must re-render as a result has to be told before the clear, not
after.

## B-15 — Four holes in the accounts change, none of which anything was doing wrong yet

**Found:** 2026-09-11, adversarial review of the accounts change before merging
**Status:** fixed 2026-09-11

Grouped because they arrived together, from reading the diff rather than from using it. Each now
has a test that fails when its fix is removed — checked by removing each fix and watching the test
go red, because a security test that cannot fail is worse than no test.

**1. Two people could both get an account.** Sign-up counted the users, then hashed a password
(~50ms), then inserted. Simultaneous requests with DIFFERENT emails all passed the count and all
inserted; the UNIQUE constraint on email only arbitrates the same address. Eight parallel sign-ups
produced **four accounts** — four separate owners of the machine. The comment in the query claiming
the constraint made this safe was simply wrong.
Fixed with `CREATE UNIQUE INDEX users_only_one ON users ((true))` (migration 00008), so every row
produces the same key and the second insert cannot land.

**2. A revoked password could still mint a session.** Sign-in verified the password and then, in a
separate call, created the session. A password change deletes every session — so a sign-in could
read the old hash, spend its 50ms while the change committed, and insert a session afterwards.
Fixed by doing both in one transaction with `SELECT … FOR SHARE` on the account row, against the
change's `FOR UPDATE`. That also stops two simultaneous changes both verifying against the same old
hash and one silently overwriting the other.

**3. A stranger could lock the owner out.** Sign-up consumed a login-throttle attempt *before*
checking whether the app was already claimed, and the throttle is keyed by client IP — which behind
a reverse proxy is one value for the whole internet. Five knocks at a closed sign-up put the
owner's sign-in into back-off. Fixed by answering 409 before touching the throttle: refusing is now
free for the server and worthless to an attacker.

**4. "Sign out everywhere" could report success having done nothing.** The server swallowed a
database error while looking up the session, and the client never checked the response at all — so
a failure cleared the screen and showed the login form, which reads as "done". That is the worst
possible lie from this particular button: it is pressed *because* a device may be in the wrong
hands, and it ended the owner's worry without ending the exposure. Fixed on both sides; a failure
now keeps the owner where they are and says "your other devices may still be signed in".

A fifth, from the same review, is D-030: revoking a session did not close the websocket it had
opened.

Two more were mine and never shipped: the 401 handler wrote a bare "signed out" into a cache slot
that now holds an object, which would have offered the *create an account* screen to somebody whose
session had merely expired; and `NewSetupCode` returned `""` on an entropy failure, with a comment
claiming that was a value nobody could match — but the submitted code is trimmed and compared in
constant time, and `"" == ""` is a match, so the one gate on claiming the app would have become
first-request-wins. It now refuses to start instead.

## B-21 — Production could never offer the Windows download

**Found:** 2026-09-13, owner: "the app can't connect to the machine"
**Status:** fixed 2026-09-13

`/install` showed a download only when `NEXT_PUBLIC_WINDOWS_INSTALLER_URL` was set at build time, but
neither `Dockerfile.web` nor Compose passed it, and no installer had been published anywhere. Every
production visitor saw "has not been published to this environment yet", so no computer could be
connected. The page now links to the stable latest-release download on GitHub, which needs no web
rebuild when a new installer is released.

## B-20 — A disconnected computer retried forever

**Found:** 2026-09-13, reading US2's daemon while tracing the same report
**Status:** fixed 2026-09-13

The server refused a revoked credential with the same 401 as one still awaiting approval, so the
daemon could not tell "stop" from "wait" and redialled on its backoff indefinitely. A revoked
credential now gets 403; the daemon deletes it and exits.

## B-19 — The Machines list never learned that a computer connected

**Found:** 2026-09-13, same trace
**Status:** fixed 2026-09-13

The `machines` event was broadcast on approve and disconnect only, not when a paired daemon
connected, dropped or reported its providers. A newly paired computer therefore stayed Offline with
no providers until the page was reloaded. All three now broadcast it.

## B-18 — The installed daemon was a console program with nowhere to log

**Found:** 2026-09-13, same trace
**Status:** fixed 2026-09-13

The launcher and the start-at-sign-in entry both started a console executable, which Windows shows
as a terminal window; closing it stopped the daemon, and its log went only to that window. US2 asks
for a computer that is reachable without a terminal. The installed daemon is now windowless, gives
the agent CLIs it starts a hidden console (Multica's approach), and logs to
`%LOCALAPPDATA%\sparstrowgen\logs\daemon.log`.

## B-17 — Windows bundle script wrote artifacts under the server directory

**Found:** 2026-09-13, US2 local package verification
**Status:** fixed 2026-09-13

`go build -C server` resolves a relative output path from `server/`, while the
archive step resolved the same path from the repository root. The resulting
bundle contained only the install script. The package script now resolves its
output directory to an absolute path before either command runs, so the archive
contains both executables and the installer.

## B-16 — A failed Postgres connection printed the database password in the server log

**Found:** 2026-09-11, deployment-readiness audit before the first Coolify deploy
**Status:** fixed 2026-09-11

**Repro:** Start the server with an unreachable `DATABASE_URL`; the `postgres unreachable` log
entry included the complete connection string.

**Expected / Actual:** The connection error identifies the failure without exposing credentials /
the log copied the full URL, including its password, into the deployment log.

**Security:** The most likely first-deploy failure is an incorrect Coolify network hostname, so
this could expose the real Postgres password precisely when the owner opens the log to diagnose it.

Fixed by logging the connection error without attaching the connection string. The pgx error still
identifies resolution, refusal and authentication failures without printing the password.
`TestPostgresFailureDoesNotLogDatabasePassword` starts the server against an unreachable dummy
database and proves its distinctive dummy password is absent from the captured output.
