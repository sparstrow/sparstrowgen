# docs/templates/

Skeletons for every kind of document in `docs/`. Copy one, fill it in, delete
the guidance comments.

**These are the canonical formats.** Where a folder's `README.md` used to
restate its own format inline, it now links here instead — one copy, not two
that drift apart.

| Creating | Template | Lands in |
|---|---|---|
| A spec — what the owner wants, in their terms | [`spec.md`](spec.md) | `docs/specs/<YYYY-MM-DD>-<slug>.md` |
| A plan — how the spec gets built | [`plan.md`](plan.md) | `docs/plans/<YYYY-MM-DD>-<slug>.md` |
| A phase spec — what a phase's tasks share | [`phase-spec.md`](phase-spec.md) | `docs/tasks/<phase>/README.md` |
| A task — one executable unit of work | [`task.md`](task.md) | `docs/tasks/<phase>/T-<phase>-<nn>-<slug>.md` |
| A verification task — proving a phase for real | [`verification-task.md`](verification-task.md) | `docs/tasks/<phase>/T-<phase>-<nn>-verification.md` |
| Feedback — raw reaction, not yet triaged | [`feedback.md`](feedback.md) | `docs/feedback/FB-<YYYY-MM-DD>-<slug>.md` |
| A bug — something behaving wrong | [`bug.md`](bug.md) | `docs/bug/BUG-<YYYY-MM-DD>-<slug>.md` |
| A security issue — a trust-boundary problem | [`security.md`](security.md) | `docs/security/SEC-<YYYY-MM-DD>-<slug>.md` |
| A runbook — steps only a human can do | [`runbook.md`](runbook.md) | `docs/runbooks/<topic>.md` |
| An entry in one of the four registers | [`register-entry.md`](register-entry.md) | appended to `OpenQuestions.md`, `Deferred.md`, `KnownGaps.md`, or `Ideas.md` |

Not sure which one? [`../README.md`](../README.md)'s "Which file does this go
in?" table answers that first — pick the destination, then the template.

**Owner-visible work starts at `spec.md`, not `plan.md`.** Anything that
changes what the owner sees, does, or can reach gets a spec first, reviewed
before planning begins. Work that only changes how the repo is built, checked,
documented, or governed goes straight to a plan with `Spec: n/a (internal)`.
The chain is spec → plan → tasks, and each links back rather than restating
the one before it.

## Conventions used in every template

- `<angle-brackets>` mark a value to replace. If one survives into a real
  document, the document isn't finished.
- `<!-- HTML comments -->` are guidance for whoever fills the template in.
  **Delete them** — they are not part of the document.
- **Delete sections that genuinely don't apply**, rather than leaving them
  with "N/A". An empty `## Traps` heading reads as "nobody looked"; no
  heading reads as "there aren't any", which is a different and more honest
  claim.
- **Dates are absolute** (`2026-08-16`), never "yesterday" or "last week".
  These files are read months later by someone who has no idea when they were
  written.
- Prose wraps at ~80 columns, matching the rest of `docs/`.

## The rule these templates exist to protect

A template is a floor, not a ceiling. It lists what a document must answer
before it counts as written — it does not cap what else you can say. If a
document needs a section no template has, add it.

What the floor buys: `AGENTS.md` §5 and `docs/README.md` both hinge on "done"
meaning the same thing every time it's written. That only holds if every task
carries a real checklist, every gap says what would close it, and every bug
says what it actually breaks. These skeletons make the required parts hard to
forget, not hard to exceed.

## Changing a template

Edit it here, and don't retro-fit existing documents. `docs/plans/` and
`docs/tasks/` are an append-only record (`docs/tasks/README.md`) — a plan written
against last month's skeleton stays exactly as it is. New documents pick up the
change; old ones are history.
