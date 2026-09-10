# Templates

Only three documents are long enough to need a skeleton. Copy the matching one, fill it in, delete
the guidance comments.

| Situation | Template | Goes to |
|---|---|---|
| "Here's how I want to use it, and what it should feel like" | [`spec.md`](spec.md) | `docs/specs/<YYYY-MM-DD>-<slug>.md` |
| "Here's exactly how we build it" | [`plan.md`](plan.md) | `docs/plans/<YYYY-MM-DD>-<slug>.md` |
| "Only a human can do this part — dashboard, DNS, secrets" | [`runbook.md`](runbook.md) | `docs/runbooks/<topic>.md` |

Everything else — ideas, deferrals, known gaps, open questions, bugs — is appended to its register
file, and each register states its own format at the top. No template, no ceremony.

**A spec has no technology in it.** No tables, endpoints, component names, or frameworks. If a
sentence couldn't be read aloud to someone who has never seen the codebase, it belongs in the plan.

**A plan is the last document before code.** There is no task layer beneath it, so it carries the
concrete steps, the files each touches, and how each is verified — enough that an agent can build
straight from it without asking anything.

Both are written with their skill (`writing-specs`, `writing-plans`), not freehand.
