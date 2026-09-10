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
