---
name: testing
description: >-
  How tests are written in this repo — where each kind lives, one canonical
  layer per behaviour, Go fixtures over open-coded inserts, and the hard rule
  that a default test must never resolve a real agent CLI from PATH. Load
  before writing or changing any test, adding a package's first test, or
  setting up the verification pipeline.
---

# Testing

No test infrastructure exists yet. These are the rules for when it does — they come from
[Multica](Reference/multica-main), which hit each of these problems at scale first.

### Never let a default test run a real agent CLI

**This is the one that matters most here.** We drive `claude`, `codex`, `agy` and `gemini`, and a
test that resolves one from `PATH` will spawn a real agent against the owner's authenticated
account and burn his quota — the exact limits this product exists to work around.

- Default tests pass a **test-created fake executable path**, or a path that deliberately does not
  exist. Never a real binary, never a bare command name.
- Real-agent smoke tests live behind a build tag and additionally check an environment variable
  before any executable lookup, so running the suite normally cannot reach them.
- Adding a new default agent command means adding it to the guard list, so ambient CLI execution
  fails the suite loudly rather than silently costing money.

### Where tests live

| What is tested | Where |
|---|---|
| Shared logic, stores, queries, hooks | `packages/core/*.test.ts` |
| Shared UI components, pages, forms | `packages/views/*.test.tsx` |
| Platform wiring — cookies, redirects, params | `apps/web/*.test.tsx` |
| Server, daemon, CLI adapters, protocol | Go tests beside the code |
| End-to-end flows | `e2e/*.spec.ts` |

Never test shared component behaviour in an app test file.

### One canonical layer per behaviour

Pure parsing, state transitions, and boundary matrices belong in a `.test.ts` beside the helper.
The component suite keeps the happy path, the wiring, accessibility, and named regressions — and
points at the canonical file in a comment. **Do not re-run a helper's matrix through a DOM mount.**

A `.test.ts` that needs no DOM starts with `// @vitest-environment node`. jsdom costs roughly
0.8s of setup per file and buys such a suite nothing. Do not add it to a test whose code branches
on `typeof window` — under node it would silently take the other path and still pass.

### Go tests build their rows through shared fixtures

Once there are DB-backed Go tests, they go through helpers in `server/internal/testutil`, not
open-coded `INSERT ... RETURNING id` with a matching cleanup, and not a
recorder/status-check/decode quartet per handler. Multica's `internal/handler` accumulated roughly
a thousand of the first and twelve hundred of the second before shared fixtures landed, and every
change to a shared contract then had to be made once per copy.

**A helper must never assert a product rule on a test's behalf.** A helper that knows what a
correct response looks like has taken the assertion away from the test making it. Keep an
assertion where the test wrote it when its message says something the shared one cannot.

### Writing them

- For a behavioural change, prefer writing the failing test in the correct package **before** the
  implementation.
- When adding an endpoint or a protocol message, add a malformed-input test alongside it.
- Anything a browser can exercise is verified in a browser as well (§4.8). A green suite is not
  evidence that a feature works.

### The verification ladder

Run the narrowest useful check while iterating; widen when the risk justifies it or when asked.
Commands are in `.sparstrowgen/blueprint.yaml`. The intended top rung is a single `make check`
running typecheck → unit tests → Go tests → end-to-end, so there is one command that means "all
of it" rather than four a person has to remember.

**Never claim a check passed unless you ran it**, and if you skipped one, say which and why.
