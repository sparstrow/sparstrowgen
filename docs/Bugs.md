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

**Found:** 2026-09-10, reviewing what is left to build **Status:** open
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
