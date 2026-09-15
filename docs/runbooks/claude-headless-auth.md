# Runbook — give the daemon a working `claude` login

**Only you can do this.** It opens a browser and asks you to approve. An agent
cannot complete it, and should not try.

## Why it is needed

The daemon spawns `claude` as a plain child process. That process reads
`~/.claude/.credentials.json`, finds an **expired OAuth access token**, and
cannot refresh it — refreshing is something the Claude Code desktop app does
(`CLAUDE_CODE_SDK_HAS_OAUTH_REFRESH=1` is set inside a desktop session, and
nowhere else).

The failure is loud but easy to misread. The CLI retries ten times, then returns
a `result` event whose `subtype` is **`"success"`** while `is_error` is `true`:

```
"result": "Failed to authenticate. API Error: 401 {...
           \"message\":\"OAuth access token has expired. Re-authenticate to continue.\"}"
```

Verified 2026-09-10. `codex` and `agy` are unaffected — both authenticate fine
from a spawned process.

## Fix

Run this in a real terminal, not through an agent:

```bash
claude setup-token
```

It issues a **long-lived** token — the mechanism intended for headless and CI
use, which is exactly what the daemon is. Then put it in the daemon's
environment:

```bash
setx CLAUDE_CODE_OAUTH_TOKEN "<the token it printed>"
```

`setx` writes it to your Windows user environment. On Windows the daemon reads
that environment afresh for every turn, so the next claude turn uses the new
token without restarting anything — however the daemon was started (docs/Bugs.md
B-28). Elsewhere, start the daemon from a **new** terminal so it inherits the
variable.

The daemon scrubs `CLAUDE_*` from the environment before spawning any CLI, and
this variable is explicitly exempted (`keepAnyway` in
`server/internal/agent/agent.go`). Do not remove that exemption — scrubbing it
would delete the fix.

## Check it worked

Send one message on claude. Or, directly:

```bash
claude -p "reply with exactly: ok" --output-format stream-json --verbose --strict-mcp-config
```

Look at the `result` line. **`is_error` must be `false`** — `subtype: "success"`
alone does not mean it worked.

## When to come back here

Whenever claude turns start failing with authentication errors and the other two
providers still work. A long-lived token can be revoked or expire; nothing about
the daemon changes, it just needs a fresh one.
