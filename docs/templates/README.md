# Templates

Two templates, because only two documents here are long enough to need a skeleton.

| Situation | Template | Goes to |
|---|---|---|
| "Here's what I want and why, and how I'd use it" | [`spec.md`](spec.md) | `docs/specs/<YYYY-MM-DD>-<slug>.md` |
| "Only a human can do this part — dashboard, DNS, secrets" | [`runbook.md`](runbook.md) | `docs/runbooks/<topic>.md` |

There is deliberately **no plan template**. What a plan used to carry now lives in the prototype's
handoff contract, derived from a design the owner has already approved — see
[`AGENTS.md` §2](../../AGENTS.md).

A spec carries **no technology and no interface design**. It says what someone needs to do and
what is true afterwards; the design step answers what it looks like. Written with `writing-specs`,
which drafts from what the owner already said rather than interviewing him for it.

Everything else — decisions, bugs, gaps, deferrals, open questions, ideas — is appended to its
register file, and each register states its own format at the top.
