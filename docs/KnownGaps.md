# Known Gaps

**What you should know before trusting an area of this system.** Read it before relying on
something, and before claiming it works.

Two kinds of entry live here, and both answer the same question — *how much can the next agent
take on faith?*

| Kind | What it is |
|---|---|
| `unproved` | We built it, but couldn't fully prove it works — or proved it works only within limits |
| `caveat` | Something noticed in passing and deliberately left alone: fragile, surprising, half-finished, or true-but-unobvious |

Neither is a bug report. If something is actually behaving **wrong**, it goes in [`Bugs.md`](Bugs.md). If it is a
question, a parked decision, or an idea, it goes in [`Later.md`](Later.md).

## When to write one

**In the same turn it surfaces**, not later — whether it came from your own work or from something
you noticed while doing something else.

- Ticked a checklist item on weaker evidence than it asked for → say so where you ticked it, *and*
  open an `unproved` entry here.
- Noticed something odd and didn't act on it because it was out of scope → open a `caveat`. Going
  back to fix it is a separate decision; recording it is not optional.

A caveat that lives only in a chat message does not exist. The next session does not read chat.

## When to close one

**Delete the entry** and say where the proof lives, or which change fixed it. The length of this
file is a real signal — a gap lingering because closing it was inconvenient is exactly the failure
this register exists to prevent.

Ids are never reused.

## Format

```
## G-n — <the claim, or the thing noticed>
**Kind:** unproved | caveat
**Raised:** <YYYY-MM-DD>, <what was being done at the time>

<What is verified and what is not — be precise about the boundary. For a caveat: what you saw,
where (file:line), and why you left it. "The platform won't emit the signal" and "nobody got round
to it" are different situations and the reader needs to know which.>

- **If wrong:** <the cost if the assumption doesn't hold. "Cosmetic and self-correcting" is a
  legitimate answer — be honest in both directions.>
- **Clears when:** <the concrete thing that closes it — an action someone can take, not "when we
  have time".>
```

---

## G-4 — What a *hit* rate limit looks like, on any provider

**Kind:** unproved
**Raised:** 2026-09-09, after closing G-1. **Narrowed:** 2026-09-09, streaming half split out to G-5.

Two related unknowns, neither closable by running anything:

1. **`claude`'s `rate_limit_event` has only ever been seen as `"status":"allowed"`.** We do not
   know the blocked value, or whether the payload changes shape when a limit is exceeded.
2. **`codex` and `agy` emitted no rate-limit signal at all** in any capture. Unknown whether they
   have one, surface it only as an error or non-zero exit when truly exhausted, or never expose it.

This is the feasibility boundary under the product's headline feature. It does **not** block the
feature: switching provider mid-conversation is user-initiated and works regardless. It blocks the
*automatic* version — "you're out on claude, switch to codex?" — and the "over your limit" UI state.

- **If wrong:** an "over limit" state gets designed against a guess and needs correcting the first
  time a real limit is hit. Worse for codex/agy: a design promising limit awareness for all three
  providers would be undeliverable for two of them.
- **Clears when:** a provider is used enough to actually hit a limit — capture the stream when it
  happens, it is the only cheap opportunity — or provider documentation describes the payload.

## G-8 — Only `agy` can enumerate its own models

**Kind:** caveat
**Raised:** 2026-09-10, feedback round item 1.

The three CLIs are not equal here, and the model lists in the app come from three different grades
of evidence:

| Provider | How the list was obtained | Grade |
|---|---|---|
| `agy` | `agy models` — prints ids and labels | **verified**, re-runnable |
| `claude` | Each documented alias run once, reading `model` back out of `system.init`: `opus` → `claude-opus-4-6`, `sonnet` → `claude-sonnet-4-6`, `haiku` → `claude-haiku-4-5-20251001` | **verified for the three aliases**; whether other ids exist is unknown |
| `codex` | No list command. Names scraped from the shipped binary's string table, cross-checked against `model = "gpt-5.6-sol"` in `~/.codex/config.toml` | **partial** — the configured one is certain, the siblings are inferred |

**`claude` does have a real discovery mechanism — our CLI is just too old for it.** A `list_models`
**control request** over the stream-json control protocol returns the catalogue without sending a
user message, so nothing is billed:

```
echo '{"type":"control_request","request_id":"x","request":{"subtype":"list_models"}}' \
  | claude --print --verbose --input-format stream-json --output-format stream-json --strict-mcp-config
```

On `claude` 2.1.90 that answers `Unsupported control request subtype: list_models` in about two
seconds and exits 0. [Multica](../Reference/multica-main) uses it in production against 2.1.223 and
2.1.258 (`server/pkg/agent/claude_models.go`), which is where this came from. **Build the adapter to
try the control request and fall back to a static catalogue** — that is Multica's shape, it needs no
version gate because an old CLI answers rather than hangs, and it upgrades itself the day the owner
updates his CLI.

`codex` has no equivalent: an invalid model returns a 400 naming no alternatives, and the valid set
is account-dependent (`"not supported when using Codex with a ChatGPT account"`).

- **If wrong:** a model offered in the picker fails at invocation time with a provider-side error.
  Recoverable and obvious, but it lands on the owner mid-conversation rather than at startup.
- **Clears when:** the daemon asks `claude` and `agy` at runtime and treats an unknown-model error
  as a reason to refresh, with a static catalogue only as the fallback. `codex` cannot be closed
  this way and stays curated.

## G-13 — Syntax highlighting covers only lowlight's `common` set

**Kind:** caveat
**Raised:** 2026-09-10, while adding the language header to code blocks (B-4)

`rehype-highlight` registers lowlight's `common` languages by default — roughly 37, including go,
python, typescript, sql, bash, java, rust, json, yaml, xml. Not included: **powershell**,
dockerfile, toml, protobuf. A block in one of those is labelled correctly and rendered as plain
text; rehype-highlight emits a `missing-language` message and moves on, so nothing is thrown and
nothing is lost.

Verified: a ```` ```powershell ```` block renders with the header "PowerShell" and zero `hljs-*`
spans (`apps/web/components/chat/markdown.tsx`).

Left alone deliberately. Registering more languages needs `highlight.js` as a direct dependency —
lowlight only re-exports it, and importing through it would be a phantom dependency — and the choice
of which to add is a question about what the owner actually writes in, not one to answer by guessing.
PowerShell is the likely first, on a Windows machine whose own commands are PowerShell.

- **If wrong:** a PowerShell or Dockerfile answer is monochrome. Cosmetic; the code is complete,
  correct, copyable and correctly labelled.
- **Clears when:** the owner says which languages matter, and `highlight.js` is added with just
  those registered.

## G-14 — What claude does when a turn contains more than one assistant message

**Kind:** unproved — **closed 2026-09-10, it was wrong. See Bugs.md B-6.**
**Raised:** 2026-09-10, after fixing the same class of bug on codex (B-5)

Cleared exactly as this entry said to: claude was run with a prompt that forces a tool call and the
`assistant` events were counted. There were four in one turn — thinking, a sentence of prose, the
tool call, then the answer — and `parseClaude` discarded every one but the last.

The predicted cost was right too. B-5 mangled text that was all still present; this deleted a whole
message with nothing on screen to suggest it. Fixed in the same change that closed this entry;
`claudeMessage.text()` now joins a message's own text blocks with a blank line as well, though that
path is still unexercised — no capture we hold has two text blocks inside one message.

## G-17 — A confirmed-clean stop cannot tell a zombie from a live process

**Kind:** caveat
**Raised:** 2026-09-11, running the process-tree tests on Linux for the first time (closing G-15)

`processTree.gone` asks `kill(-pgid, 0)` on Unix, which succeeds for any process in the group that
still has a PID — **including one that has already exited and is waiting to be reaped**. A zombie is
dead and holds nothing open, but it answers that question exactly as a running process does.

Normally invisible: an orphaned process is reparented to init, which reaps it immediately, so the
zombie window is microseconds. It appeared the moment these tests ran in a container, where
`go test` was PID 1 and PID 1 does not reap orphans — a stop that had genuinely killed everything
reported "could not be confirmed clean" after 6.5 seconds, against 0.5 with an init present.
`make test-linux` passes `--init` for exactly this reason.

**The kill is unaffected.** This is the confirmation, not the killing: nothing leaks, the report is
wrong rather than the outcome.

- **If wrong:** a spurious warning in the daemon log claiming a stop may have left something
  running when it did not. Cosmetic on a normal host; noisy on a daemon deployed into a bare
  container with no init, which is plausible given the server is headed for Coolify.
- **Clears when:** either the daemon is only ever run under an init (worth asserting when it is
  packaged), or `gone` reads `/proc/<pid>/stat` and treats state `Z` as gone. The second is
  Linux-only and would need a different answer on macOS, which is why it was not done now.

## G-32 — The Windows installer is unsigned and not yet proved on a clean computer

**Kind:** unproved
**Raised:** 2026-09-13, US2 implementation · **Revised:** 2026-09-13, when the installer became one
self-installing executable (D-033), and again after the owner's first real install

**Verified.** `scripts/package-windows.ps1` builds `sparstrowgen-setup.exe` for one server. API tests
cover the pairing journey against a real database: claim once, refused before approval (401),
connected after approval, providers and the `machines` event reaching the browser, 403 after
disconnect while another computer stays online, and another account unable to see, approve or
disconnect. Daemon tests cover the link parser, claiming and saving a credential, a release ignoring
leftover `SERVER_WS` and `DAEMON_TOKEN`, and a 403 deleting the credential and stopping. The release
executable itself was run on the development PC without installing: background mode with no
credential logged "not paired yet" and exited without a window, and a made-up pairing link was
refused by production, saved nothing, and showed an error box.

**Proved on the owner's PC, 2026-09-13,** with a locally built production executable (SHA-256
`3a8441d4…8922`) against `api.sparstrow.com`: the installer copied itself into
`%LOCALAPPDATA%\Programs\sparstrowgen` and registered `sparstrowgen://` and start at sign-in; **Add
computer** in Chrome opened the link, which claimed the request; the daemon was refused four times at
two-second intervals until approval, connected nine seconds after the claim, reported claude, codex
and agy as available, and ran one real Chat turn (claude, 4 s). One copy ran, with no visible window.

**Not verified.** Still connected after signing out of Windows and back in; installing over a running
copy; the journey on a Windows account that has never run sparstrowgen; the download from the
published release rather than a local build. The executable is unsigned, so Windows SmartScreen
warns before running it.

- **If wrong:** a first-time user downloads the installer and still cannot pair, with the browser
  showing "has not answered yet".
- **Clears when:** all of the following pass and the proof is recorded here:
  1. After signing out of Windows and back in, the owner's paired computer is online without anyone
     starting anything.
  2. On a Windows account that has never run sparstrowgen, the published download installs, pairs
     from Machines, and runs one Chat turn.
  3. A Windows code-signing identity signs the executable in the release step, and the published
     signature verifies.

## G-33 — Reinstalling or re-pairing ends turns running on that computer

**Kind:** caveat
**Raised:** 2026-09-13, building the self-installing executable

Installing over a running copy, or opening a new pairing link, asks the running background daemon to
exit so the new copy or credential takes over (`stopRunning` in `server/cmd/daemon/install_windows.go`).
Any agent turn in flight on that computer ends, and the conversation records that the computer
disconnected; resending it works. Left alone because both are deliberate actions by the person at
that computer, and waiting for idle is exactly what US3's updater must build properly rather than
something to half-build here.

- **If wrong:** a long turn is lost when someone re-pairs mid-answer.
- **Clears when:** US3's idle-aware activation is used for reinstall and re-pairing too.

## G-19 — How Coolify treats the one-shot migration container across redeploys is unverified

**Noticed:** 2026-09-11, codex reviewing the deployment artifacts before the first deploy

`docker-compose.yaml` runs migrations as a `migrate` service with `restart: "no"`, and the server
waits on `condition: service_completed_successfully`. That ordering is **verified**: running this
compose file locally against a real Postgres, `migrate` ran, exited 0, and only then did `server`
start.

What is not verified is what Coolify v4.3.18 does with an *exited* container on the **second**
deploy — whether it recreates it, reuses it, or treats a zero-exit container as a failed service
and reports the deployment unhealthy. That is Coolify lifecycle behaviour, and nothing short of
deploying twice will answer it.

Two related things also unproved, and worth knowing before they surprise someone:

- The dependency orders **startup**, and does not stop the previously-running API while migrations
  run during a redeploy. A migration that breaks compatibility with the old server would be applied
  underneath it.
- A migration that fails leaves the old server running and the new one never starting, which is the
  safe direction but will look like "the deploy hung".

**Closes when:** the second deploy to Coolify either works or does not. If it does not, the fallback
is a Coolify pre-deployment command rather than a compose service.

## G-21 — The chat surface does not work at phone width

**Noticed:** 2026-09-11, verifying the account screens in a browser

At 375×812 the sidebar keeps its fixed 288px and the transcript is squeezed into what is left, so
the conversation pane wraps to one or two words a line and the account menu opens off-screen. The
sign-in and sign-up screens are fine — they are a centred column — so this is the chat surface
only, and it predates the account work.

Deliberately not fixed here. It needs a real decision about what the sidebar does on a phone
(drawer, or a list-then-detail view), and that is a design question for the owner rather than a
CSS patch. Nothing about it is a surprise once seen, which is why it is written down rather than
guessed at.

**Closes when:** the chat surface has a phone layout the owner has chosen.

## G-22 — Coolify v4.3.18 imports a Compose required-value message as the value

**Noticed:** 2026-09-12, first production deployment to Coolify v4.3.18

The initial Compose definition used expressions such as
`${DATABASE_URL:?set this to the Postgres resource internal URL}`. Coolify
created an environment variable whose value was the text after `:?`, then
started the stack. The `migrate` container therefore received that sentence as
`DATABASE_URL`; Goose exited with `cannot parse ... as keyword/value`, and the
`service_completed_successfully` dependency correctly prevented `server` and
`web` from starting.

The Compose file now uses message-free `${VAR:?}` expressions and the deploy
runbook tells the operator to inspect every generated value. This existing
Coolify resource still retains its imported placeholder until it is manually
replaced because reloading Compose preserves saved variable values.

**Closes when:** creating a fresh v4.3.18 Compose resource from the corrected
file leaves required variables empty and blocks deployment before starting a
container.

## G-24 — Coolify Compose interpolation needs `API_ORIGIN` in its runtime `.env`

**Noticed:** 2026-09-12, first automatic production deployment after changing
the Compose port declarations

`API_ORIGIN` is referenced only under `web.build.args`, so it was configured as
build-time-only. Coolify v4.3.18 nevertheless runs `docker compose pull` using
the runtime `.env` before building images. Compose interpolates build arguments
at that stage and failed with `required variable API_ORIGIN is missing a value`.

The Coolify variable must therefore have both Build time and Runtime enabled.
This does not add it to a running container: the Compose file uses the value
only in `web.build.args`; it is a public API origin rather than a secret.

**Closes when:** Coolify separates Compose interpolation variables from
container runtime variables, or its documentation describes this requirement.

## G-25 — Coolify labels the Compose resource “Running (no healthcheck)”

**Kind:** caveat
**Raised:** 2026-09-12, verifying the first successful production deployment

The live v4.3.18 resource displayed **Running (no healthcheck)** at application
level even though `server` has a Docker healthcheck in `docker-compose.yaml`.
Both public routes worked, account creation completed through the API, and the
server log showed it listening on `0.0.0.0:8080`; the label therefore did not
mean this deployment lacked all operational proof. It appears to describe the
aggregate Compose resource rather than the individual service, but that UI
interpretation has not been verified from Coolify itself.

- **If wrong:** an operator may trust the green/running label as evidence of API
  health, or treat its “no healthcheck” suffix as evidence that Compose ignored
  the service check. Either inference is stronger than what the UI proved.
- **Clears when:** Coolify shows the individual service health state, or its
  v4.3.18 behavior/documentation establishes exactly what the aggregate label
  represents. Until then, use `/api/health` and the service runtime log.

## G-26 — The deployment runbook has not been re-verified on Coolify v4.3.19

**Kind:** caveat
**Raised:** 2026-09-12, reviewing the proposed production/staging pipeline

The first production deployment and `docs/runbooks/deploy.md` were verified on
Coolify v4.3.18. The project dashboard now reports v4.3.19. No changed behavior
has been observed yet, but the version boundary is real: the environment-variable,
Compose parsing, network and domain screens are instructions for v4.3.18 until
the staging application exercises them on v4.3.19.

- **If wrong:** a click path or one of the v4.3.18-specific workarounds in the
  runbook may be stale. Production is already running, so the immediate cost is
  misleading setup guidance rather than an outage.
- **Clears when:** the staging application is configured and deployed on
  v4.3.19, each affected runbook step is checked against its actual behavior,
  and the runbook version note is updated.

## G-30 — Behind the proxy, "per client" throttling is per proxy

**Kind:** caveat
**Raised:** 2026-09-12, adding access requests

`clientIP` deliberately reads `RemoteAddr` rather than a spoofable `X-Forwarded-For`, so on Coolify
every visitor is the proxy's address. Sign-in and access requests have separate counters, which
keeps a stranger flooding requests from locking the owner's sign-in — but a flood of requests does
slow down other people's requests, and repeated wrong passwords from anyone slow down everyone's
sign-in. Per-address spacing of emails is unaffected: it is keyed by recipient.

- **If wrong:** at invitation-only scale, a nuisance rather than an outage: "too many attempts —
  try again in Ns" for a real person while someone else misbehaves.
- **Clears when:** the proxy's forwarded address is trusted only from the proxy's own IP (Coolify's
  Traefik network), and both throttles key on that.
