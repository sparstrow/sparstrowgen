<!--
TEMPLATE — copy to docs/security/SEC-<YYYY-MM-DD>-<slug>.md, then delete every
HTML comment in the copy.

THE RULE: document a security concern in the SAME TURN it surfaces —
owner-reported or agent-found — rather than relying on chat history being
re-read.

⚠️ NEVER put a live secret, a real token, or a replayable exploit payload in
this file. These are committed to git, so writing the value here turns a
report into a second copy of the exposure. Describe the CLASS of exposure and
where it lives — file and line, the env var's name — not the value itself.

If a real secret was actually exposed, rotating it is an owner action: record
the FACT here and add a row to docs/runbooks/README.md.

After filing: add a row to docs/security/README.md's index, and open a task in
docs/tasks/ linked back to this file's id. High and critical severity should
generally jump the queue ahead of whatever is in flight — say so explicitly
when opening the task.
-->

# SEC-<YYYY-MM-DD>-<slug>

**Status:** <🔴 open | 🟡 investigating | 🟢 resolved>
**Severity:** <critical | high | medium | low>
**Reported by:** <owner | agent — name what you were doing when you found it>
**Reported:** <YYYY-MM-DD>

## What's exposed / what's possible

<!--
The concrete thing an attacker — or an over-privileged legitimate user — could
do, and its effect. An action and an outcome.

NOT "this feels insecure" and not "this endpoint lacks validation". Write the
capability: "any authenticated user can read every workspace's runs by passing
another workspace's id, because the handler filters on the client-supplied
value rather than the session's membership."
-->

## Who can trigger it

<!--
Anonymous internet · any authenticated user · a specific role · only a
workspace member · only with local or physical access · only the service role.

This is most of what severity depends on, so be precise. The gap between "any
authenticated user" and "only a workspace owner" is usually the gap between
critical and low.
-->

## Evidence

<!--
What was checked and how it was confirmed — file and line, an auth check read
that shows the gap, a request that demonstrates it, a schema fragment.

Describe the request; do not paste a working exploit. "GET /api/v1/runs with
another workspace's id in the query returns its rows" is evidence. A
copy-pasteable curl with real ids and a real token is a weapon in the repo.

If it is suspected rather than confirmed, say which it is — an unconfirmed
report filed honestly is useful; one written with false confidence sends
someone chasing a phantom.
-->

## Impact

<!--
Worst case if left unfixed, and the blast radius: whose data, how much, and
whether it is recoverable.

Say whether it is exploitable TODAY or only under some other condition — a
public deployment, a second user in a workspace, a provider that isn't enabled
yet. A vulnerability gated behind something that hasn't happened is real, but
it is a different urgency, and the gate is exactly the thing that will
silently disappear later.
-->

## Resolution

<!--
Filled in when closed: the fix, where it landed (commit/PR), and HOW IT WAS
VERIFIED CLOSED — not "should be fixed now".

For this folder specifically, verification means re-running the thing that
demonstrated it and observing the refusal. A security issue closed on
inspection alone is a security issue closed on a hunch; if that is genuinely
all that was possible, say so and open a KnownGaps.md entry.

Leave the file in place after closing and flip Status to 🟢. This folder is a
record, not a queue that empties out.
-->
