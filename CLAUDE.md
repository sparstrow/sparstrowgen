# CLAUDE.md

**Read [AGENTS.md](AGENTS.md) first** — it holds the workflow, git rules, and engineering
standards for every agent working here. `.sparstrowgen/blueprint.yaml` holds the stack.
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

**Division of labour:** he supplies user scenarios and the experience he expects; agents decide the
implementation. Don't hand library-level choices back to him as open questions — recommend, explain
why, and let him veto. When something genuinely is his call, use the options framework in
[AGENTS.md §4](AGENTS.md).

**Frontend and user experience are the top priority**, with a lot of customization expected. Give
interface decisions more depth and more options than backend ones. That is where he wants to be
involved; the backend he largely wants decided well on his behalf.

## Choosing approaches

**Default to the long-term correct option, not the fast one.** Do not build something knowing it
will be thrown away and rewritten. In his words: *"I would only choose fast option for testing or
proof of concept, not when I know it will need to be replaced."*

- Lead with the durable approach. Mention a faster alternative as a rejected option with the
  reason, not as an equal choice.
- A fast path is fine for a spike — say so out loud when proposing it, so its throwaway status is
  agreed rather than assumed.
- Spend extra care on schema, protocol, and boundary decisions. Those are the expensive ones to
  retrofit.

## Build rhythm

Small steps, one feature at a time, each tested end to end before the next begins. This is a
rebuild — an earlier attempt at sparstrowgen was abandoned, and the lesson taken from it was to
build in verifiable increments rather than broad parallel layers. AGENTS.md §3.6 is the rule;
this is why it exists.
