<!--
TEMPLATE — copy to docs/runbooks/<topic>.md, then delete every HTML comment in
the copy.

A runbook is for steps ONLY A HUMAN CAN DO: an account only the owner
controls, a dashboard setting, a secret only they should type. If an agent
could do it, it is a task, not a runbook.

Runbooks are not a lifecycle stage. They don't graduate into code — they sit
in docs/runbooks/ as reference for as long as the manual step exists.

⚠️ TWO THINGS TO GET RIGHT:

1. Add a row to docs/runbooks/README.md. That file is the owner's single
   action-item checklist and the ONLY place a status lives. Everywhere else
   points at it. A status written in two places is the exact drift that file
   exists to prevent.

2. No live secrets in this file. Name where a value comes from — "Coolify →
   Service → Environment Variables" — never the value itself.

The audience is the owner, not an agent. Write for someone who has not been
in this codebase today: name the dashboard, the menu path, and the button.
-->

# <What this achieves — e.g. "Deploying the web app, and pointing the desktop shell at it">

**Why this needs you:** <the account, secret, or dashboard access an agent
cannot have>

**What is already done:** <the code side, so the reader knows they are not
being asked to build anything — name the task or phase that shipped it>

---

## What "<not done yet>" currently means

<!--
DELETE once the runbook has been followed.

The consequences of the current state, especially the invisible ones. This is
what stops someone concluding it doesn't matter: "every daemon defaults to
localhost:3000, so pairing works today only because the app happens to be
running on the same box" is the kind of thing that is obvious in hindsight and
invisible in advance.
-->

## 1 — <first step>

<!--
Numbered steps, each ending in something observable. Include:

  - the exact menu path (Coolify → Service → Environment Variables)
  - which values are safe to expose and which are server-only
  - the mistake people actually make at this step, and what it looks like
    when made — "this is the proxy's callback URL, not the
    app's; registering the app URL fails with a redirect-URI mismatch" is the
    model here

Env vars in a table: variable, where it comes from, and any warning.
-->

| Variable | Where it comes from | Notes |
|---|---|---|
| `<NAME>` | <dashboard path> | <safe to expose / server-only> |

## 2 — <next step>

## Check it

<!--
How the owner knows it worked, as ticks. Concrete and observable — a page that
loads, a row that turns green, a curl whose output changes.

If a check can only be done later, or by an agent, say which and point at the
task that covers it.
-->

- [ ] <observable result>

## Notes

<!--
DELETE if empty. Things that are true and surprising: how identities link
across providers, what happens on a second run, what this does NOT unblock.

"What this does not unblock" is worth stating explicitly whenever the reader
might reasonably expect it to. Someone who deploys the app will assume
terminals now work in the desktop window; saying otherwise here saves a bug
report.
-->
