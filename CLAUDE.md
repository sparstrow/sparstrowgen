# CLAUDE.md

**Read [AGENTS.md](AGENTS.md) first** — §1 is how work happens here, and it is unusual enough that
skipping it will send you the wrong way. `.sparstrowgen/blueprint.yaml` holds the stack.
[README.md](README.md) holds the folder layout.

This file covers one thing those don't: how to work with the owner.

## Working with the owner

Business systems analyst by profession — works with Microsoft Dynamics NAV and EDI, and spends his
days advising departments on new procedures and walking them through process changes. That shapes
how to communicate here.

**Technical depth is medium and unevenly distributed:**

| Speak freely | Explain plainly, with the tradeoff stated |
| --- | --- |
| APIs, client/server architecture, how they communicate | Specific library choices |
| SQL queries, general data modelling | Embeddings and vector search |
| App structure, product and UX patterns | Infrastructure and database internals |

Strong product intuition from having used a lot of apps, even where implementation knowledge is
thinner. Never condescend, never assume.

**Division of labour:** he supplies scenarios and judgment; agents decide implementation. Don't
hand library-level choices back to him as open questions — recommend, explain why, and let him
veto.

## He works visually — show, don't tell

**Frontend and user experience are the top priority**, with a lot of customization expected. He
described the way he wants to work as *pure vibe coding*: ask for something, see designs, pick
one, watch it get wired into the real app, then have the backend built underneath it. He liked
that a lot, and [AGENTS.md §1](AGENTS.md) exists to make it the default.

What this means in practice:

- **If you can render it, render it.** Two or three genuinely different directions beat any amount
  of prose describing them. Seeing them *is* the comparison.
- **He judges best in context** — a real route in the real app on placeholder data tells him more
  than a standalone mockup.
- Backend decisions he largely wants decided well on his behalf. Design decisions he wants to
  make himself, from options.

## Choosing approaches

**Default to the long-term correct option, not the fast one.** Do not build something knowing it
will be thrown away. In his words: *"I would only choose fast option for testing or proof of
concept, not when I know it will need to be replaced."*

This is not in tension with moving fast. What he rejects is **paperwork**, not engineering care:

- Lead with the durable approach. Mention a faster alternative as a rejected option with the
  reason, not as an equal choice.
- Spend real care on schema, protocol, and boundary decisions — those are expensive to retrofit,
  and they go in [`docs/Decisions.md`](docs/Decisions.md).
- Spend as little as possible on documents. A previous attempt at this app was abandoned without
  shipping because the budget went into planning instead of code: *"nothing was shipped at all."*
- A fast path is fine for a spike — say so out loud, so its throwaway status is agreed rather than
  assumed.

## The failure mode to watch for

Two attempts at this app now. The first died of process. The pull toward "let me just write a
document about this first" is real and it is the thing to resist — including for me, which is why
`AGENTS.md` §4 opens with rule zero.

If a document isn't going to change what gets built, don't write it. Show him something instead.
