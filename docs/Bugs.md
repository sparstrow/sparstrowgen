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
