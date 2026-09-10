<!--
TEMPLATE — copy to docs/bug/BUG-<YYYY-MM-DD>-<slug>.md, then delete every HTML
comment in the copy.

THE RULE: document a bug in the SAME TURN it surfaces — owner-reported or
agent-found — rather than relying on chat history being re-read. A bug
mentioned only in a chat message does not exist to the next session.

Is it a security issue instead? If the wrong behavior lets someone do
something they shouldn't — auth bypass, injection, credential exposure, data
crossing users or workspaces, a leaked credential — use security.md instead. When in
doubt, file it there: a false positive costs a few minutes, a missed one
doesn't.

After filing: add a row to docs/bug/README.md's index, and once the bug is
understood well enough to fix, open a task in docs/tasks/ linked back to this
file's id. The report stays as the historical record; the task is what gets
executed.
-->

# BUG-<YYYY-MM-DD>-<slug>

**Status:** <🔴 open | 🟡 investigating | 🟢 resolved>
**Reported by:** <owner | agent — name what you were doing when you found it>
**Reported:** <YYYY-MM-DD>

## Symptom

<!--
What actually happens, from the user's or the system's point of view.
Concrete — inputs, steps, observed output.

NOT a guess at the cause. The single most common failure here is writing
"the session token isn't being refreshed" when what you observed was "I got
logged out after a few minutes". Write the second one; the first belongs in
Investigation, once it's evidence rather than a hunch.
-->

## Reproduction

<!--
Exact steps, numbered, starting from a known state. Include what you expected
and what happened instead.

If it can't be reproduced on demand, say so and give the exact evidence you
do have — the log line, the screenshot, the row in the database, how many
times out of how many attempts. "Intermittent" with no evidence is not a bug
report.
-->

## Investigation

<!--
What was checked, what was ruled out, and what is still suspected. Update this
section as it develops rather than starting a second file.

Per AGENTS.md §3.2, base diagnoses on log evidence — read the full,
un-truncated stack trace before proposing a cause. Note what you ruled out and
how; the next person then doesn't repeat it.
-->

## Impact

<!--
What breaks and for whom, if left unfixed. Who hits it, how often, and whether
there's a workaround.

This is what decides priority, so be honest in both directions — a cosmetic
bug described as severe distorts the queue as much as the reverse.
-->

## Resolution

<!--
Filled in when closed: root cause, the fix, and where it landed (commit/PR).

"Where it landed" means a real reference — a commit sha, a PR link, a file
path. "Fixed in the auth refactor" is not findable six months from now.

Say how it was verified closed, not just that it should be. Leave the file in
place after closing and flip Status to 🟢 — like KnownGaps.md, this folder is
a record, not a queue that empties out.
-->
