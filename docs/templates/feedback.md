<!--
TEMPLATE — copy to docs/feedback/FB-<YYYY-MM-DD>-<slug>.md, then delete every
HTML comment in the copy.

THE RULE: capture feedback in the SAME TURN the owner gives it, verbatim,
before triaging it. Paraphrasing while capturing loses exactly the detail
that later separates "annoying" from "broken" from "would be nice someday".
A feedback item mentioned only in a chat message does not exist to the next
session.

This file is a holding pen, not a destination. Every item eventually routes
to one of: docs/bug/ (it's behaving wrong), docs/security/ (trust-boundary),
docs/Ideas.md (unscoped, no commitment), docs/specs/ (a real feature/change
worth building), or Won't Fix (routed nowhere, with a reason). See
docs/feedback/README.md for the full routing rule.

After filing: add a row to docs/feedback/README.md's index.
-->

# FB-<YYYY-MM-DD>-<slug>

**Status:** <🔴 new | 🟡 triaged | 🟢 routed>
**Reported by:** <owner — or name the person if feedback arrives secondhand>
**Reported:** <YYYY-MM-DD>
**Area:** <page/feature this is about — e.g. "Chat", "Provider switching", "Preview", or "general">

## Raw feedback

<!--
Verbatim. Paste what the owner actually said, unedited. If it arrived over
several messages, include all of them in order. Do not summarize here — the
Context section below is where you interpret it.
-->

## Context

<!--
What prompted this, if known: what the owner was doing, what they expected,
what surprised them. Fill in from the conversation it arrived in, not from
guessing at intent. Leave blank (delete the heading) if the raw feedback is
already fully self-contained.
-->

## Triage

<!--
Filled in once looked at — doesn't have to happen the same turn as capture,
but should happen before the item is considered actioned.

State which bucket this belongs to and why:
- Bug → link the new docs/bug/BUG-<date>-<slug>.md
- Security → link the new docs/security/SEC-<date>-<slug>.md
- Idea (unscoped, no commitment) → link the new docs/Ideas.md entry
- Worth building → link the new docs/specs/<date>-<slug>.md (or an existing
  spec/plan/task this now feeds into)
- Won't fix / already covered / out of scope → say why, and by what
  (an existing KnownGaps.md entry, a deliberate product decision, etc.)

One feedback item can route to more than one place (e.g. "this is broken AND
suggests a real feature gap") — list every destination it produced.
-->

## Resolution

<!--
Filled in once every routed destination has landed (the bug is fixed, the
spec is built, the idea is explicitly declined). Say where — commit/PR/task
id, not "handled elsewhere". Leave the file in place after closing and flip
Status to 🟢 — like docs/bug/, this folder is a record, not a queue that
empties out.
-->
