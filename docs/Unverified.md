# Unverified

**Checks nobody has run yet, on things that are built.** A change can ship before every part of it
has been watched working — a sign-out, a second Windows account, a real limit being hit. Each such
check gets an entry here, in the same change, so it is picked up later instead of forgotten.

This file is the to-do list of checks. [`KnownGaps.md`](KnownGaps.md) keeps the reasoning — why
something is uncertain and what it costs if it is wrong — and points here for the checks that would
settle it. [`Bugs.md`](Bugs.md) is for a check that ran and failed.

## Working the list

- **Add** an entry whenever you report something done without having seen part of it work. Name the
  check, not the worry: something a person can do, and what passing looks like.
- **Verify when time permits.** Before finishing a turn, look at the open entries. Run any the agent
  can run now — a browser, the testing account, the local stack, a log on the owner's PC. When the
  owner is already testing, hand him the ones only he can run, as steps.
- **Close** an entry by changing its status in place, with the date and one line of proof. Never
  delete one. A check that fails becomes a bug; link it.
- **Never mark verified on weaker evidence than the check asks for.** Say what was actually seen.
- When every entry a gap points at is verified, close that gap too.

Ids are never reused.

The owner's open checks, as one sitting in the order to run them:
[`runbooks/owner-checks.md`](runbooks/owner-checks.md).

## Format

```
## U-n — <the check, as something someone can do>
**From:** <change, PR or gap>, <YYYY-MM-DD> · **Who can run it:** agent | owner | needs <what>
**How:** <steps, and what passing looks like>
**Status:** open | verified <YYYY-MM-DD> — <proof> | failed <YYYY-MM-DD> — <B-n>
```

---

## U-1 — A paired computer comes back online after signing out of Windows and in again

**From:** US2 pairing (#11), G-32, 2026-09-13 · **Who can run it:** owner, then agent reads the log
**How:** With the computer shown Online in Machines, sign out of Windows and sign back in. Start
nothing. Within a minute Machines shows it Online, and `%LOCALAPPDATA%\sparstrowgen\logs\daemon.log`
has a `connected` line after the sign-in time, from a process started at sign-in.
**Status:** open

## U-2 — A Windows account that has never had sparstrowgen installs, pairs and runs a turn

**From:** US2 pairing (#11), G-32, 2026-09-13 · **Who can run it:** owner — needs a second Windows
account or a clean PC
**How:** On that account, download from the install page, open the installer (SmartScreen: More info,
Run anyway), choose Add computer in Machines, approve, then send one Chat message. Passing: the
computer shows Online with its providers and the message gets an answer.
**Status:** open

## U-3 — Not now on a real pairing leaves nothing behind

**From:** #12, 2026-09-13 · **Who can run it:** owner (needs a computer opening the link)
**How:** In Machines choose Add computer, let the computer answer, then choose **Not now**. Passing:
the panel closes at once, Machines lists no new or waiting computer after a refresh, and the
computer's log shows the pairing refused and the earlier connection, if any, still working.
**Status:** open — everything but the panel verified 2026-09-14 on production with the testing account
and test computers on this PC (CLAUDE.md §1). A fresh computer claimed a pairing and it was declined
through the endpoint Not now calls (204). The pairing now reads `rejected`, Machines still listed only
the two computers already paired, both still online, and the computer's log said "this pairing was
declined or has expired; keeping any earlier pairing" before it exited with its pending credential
removed. **Still to see:** the panel closing at once. Opening Add computer in the browser launches the
installed copy through its link, which would re-pair the owner's PC, so only he can click it.

## U-4 — Add computer on an already connected computer changes nothing

**From:** #12, 2026-09-13 · **Who can run it:** owner
**How:** On the connected PC, choose Add computer. Passing: it says "This computer is already
connected", Machines still lists exactly one entry for it, and it stays Online without reconnecting.
**Status:** open — everything but the browser's wording verified 2026-09-14 on production with the
testing account. A connected test computer claimed a new pairing and printed "this computer is already
connected to that account; nothing changed", keeping its credential and writing no pending one. The
pairing reads `approved` with that computer's existing id, Machines still listed exactly the same two
entries, and its log gained no "connection lost" or second "connected" line. **Still to see:** the
panel saying "This computer is already connected", for the same reason as U-3.

## U-5 — A disconnected computer is refused and needs approval to come back

**From:** US2 pairing (#11), 2026-09-13 · **Who can run it:** owner (it ends his pairing until he
pairs again)
**How:** Open the computer in Machines and disconnect it. Passing: it leaves the list, its log shows
it was disconnected and stopped, `machine-credential` is gone from `%LOCALAPPDATA%\sparstrowgen`,
and only a new Add computer plus approval brings it back.
**Status:** verified 2026-09-14 — on production with the testing account and two test computers on this
PC, not the owner's. Disconnecting test computer 1 (204) left only test computer 2 listed. Computer 1's
log said "this computer was disconnected from its account; forgetting its credential", its data folder
was empty and its process had stopped; started again it exited with "no machine credential is
available". Meanwhile computer 2 ran "Reply with exactly: ok" on claude to "ok" (22:50:54–22:50:57).
Computer 1 then claimed a new pairing and, before approval, was refused every two seconds with "the
server has not accepted this computer: it is waiting for approval in the browser"; approved at
22:52:32, it connected at 22:52:33 with "this computer's pairing was approved".

## U-6 — The owner confirms Settings → Updates reads and behaves as he wants

**From:** US3 (#15), 2026-09-13 · **Who can run it:** owner
**How:** He chose the design by pointing at Multica's Updates page and approved building the rest
without confirming it in the app first. Open Settings → Updates, try Check now and the automatic
switch, and say what to change. Passing: no changes, or his changes recorded as feedback.
**Status:** open — his judgement, which no test replaces. Seen 2026-09-14 on production with the testing
account and two development-build test computers: each card said Online, Current version "Development
build", Automatic updates "Available once this computer has a version that can update itself", and
"This version cannot update itself. Download the installer and open it on this computer." with Download
update. Check now and the switch only work on an installed release, and a second installed copy cannot
run beside his under the same Windows user (they share `Local\sparstrowgen-daemon`).

## U-7 — An installed computer updates itself from a published release and stays connected

**From:** US3 (#15), D-034, 2026-09-13 · **Who can run it:** agent on the owner's PC (it needs a
published release, and the installed copy)
**How:** With 0.2.0 installed and connected, publish a higher release through
`docs/runbooks/daemon-release.md`. Within the hour, or at once with Check now, the log shows the
check, `handing over to the updater`, the updater's `updated`, and a `connected` line from the new
version; Machines shows it Online with the same pairing, and Settings → Updates shows the new version.
**Status:** verified 2026-09-13 — on the owner's PC, 0.2.0 installed from the published download
connected at 23:47:13. Its first automatic check found `daemon-v0.2.1`, downloaded and verified it
(23:48:14) and handed over at 23:48:15; the new copy connected at 23:48:16 with the same credential
and reported its providers, and the updater logged `updated` 0.2.0 → 0.2.1 at 23:48:21. The installed
executable's SHA-256 is 0.2.1's (`20c30ccc…2873`), one copy runs, and no failure was recorded.
Machines and Settings → Updates were not looked at: they need his sign-in.

## U-8 — An update that cannot reconnect is put back on a real computer

**From:** US3 (#15), D-034, 2026-09-13 · **Who can run it:** agent — needs a release channel that is
not the one every installed computer follows (a staging channel, WORKFLOW.md phase 2) or a test
build pointed at a private manifest
**How:** Publish, to that channel only, a build that cannot reach the server. Passing: after
two minutes the log shows the updater putting the old version back, the old version connects, and
Settings → Updates shows "did not reconnect within 2 minutes, so v… was put back and is running".
The runUpdate tests prove the sequence with fake processes; this proves it with real ones.
**Status:** open — the rollback itself verified 2026-09-14 on the owner's PC, with his approval. A test
base build 0.9.1, reading updates from a pre-release `daemon-test-rollback` that no real computer
follows, was put in place with `apply-update` at 20:46:06 (from 0.2.3). Its first check found 0.9.2, a
build pointed at `wss://127.0.0.1:9`, and handed over at 20:47:03. 0.9.2 was refused 14 times, and at
20:49:04 the updater logged "v0.9.2 did not reconnect within 2 minutes, so v0.9.1 was put back and is
running". 0.9.1 connected the same second, with its hash back in place (`aa23de8b…`), `result.json`
read, and one copy running. Its next check, at 20:50, did not retry 0.9.2. The PC was then returned to
official 0.2.3 the same way (`updated` 0.9.1 → 0.2.3 at 20:50:47, hash `ac302635…480f`), and the test
release and its tag were deleted. **Still to see:** that message in Settings → Updates on his account,
which needs his sign-in; the restored status was replaced when 0.2.3 went back in.

## U-9 — An update waits while a real agent turn runs, and installs when it ends

**From:** US3 (#15), 2026-09-13 · **Who can run it:** owner (it needs his signed-in Chat)
**How:** Start a long Chat turn, then press Update now in Settings → Updates while it runs. Passing:
it says "It installs after the agent task running on this computer finishes", the turn completes
normally, and only then does the computer install and reconnect.
**Status:** open

## U-10 — Chat on a too-old computer says to update it instead of failing

**From:** US3 (#15), 2026-09-13 · **Who can run it:** agent — needs a deployment where
`MinDaemonProtocol` can be raised without stranding real computers (staging)
**How:** Raise `MinDaemonProtocol` above a connected computer's protocol. Passing: Chat's composer
says the computer is too old and to update it in Settings → Updates, sending is disabled, and
Settings → Updates marks it too old with Download update. `TestATooOldComputerIsToldToUpdateAndSentNoWork`
proves the server side; the surface has not been seen in this state.
**Status:** verified 2026-09-14 — on a local stack rather than staging: main's server built in a scratch
copy with `MinDaemonProtocol = 2` (port 8091), the web app on 3091, and a dev daemon (protocol 1)
paired through the real pairing flow, with its own `SPARSTROWGEN_HOME` so the PC's credential was never
read. `/api/machines` returned it online with `tooOld: true`. Settings → Updates said "Too old for
sparstrowgen. This computer cannot run agent work until it is updated", with automatic updates
unavailable and Download update. Chat marked claude, codex and agy "Computer needs an update" and said
"Your computer's sparstrowgen is too old for this app, so nothing new can be sent. Update it in
Settings → Updates. Everything already said stays readable."; the message box and Send message were
both disabled, and the console had no errors. The test session and computer were deleted afterwards.
Not seen as a screenshot: the browser pane would not draw, so page text and DOM state were read.

## U-11 — Settings → Updates shows its error card when the computers cannot be loaded

**From:** US3 (#15), G-36, 2026-09-13 · **Who can run it:** agent
**How:** With the API stopped, open Settings → Updates. Passing: within a few seconds it says
"Updates could not be loaded" and that nothing on the computers has changed, and Try again recovers
once the API is back.
**Status:** verified 2026-09-14 — on production with the testing account, in a visible Browser pane,
with the machines request made to fail in the page as a refused connection does (the API itself
cannot be stopped there). The loading placeholders showed, then after 1,010 ms "Updates could not be
loaded" with "Nothing on your computers has changed." and Try again; with requests allowed again,
Try again showed the computer within 250 ms. The 2026-09-13 attempt stayed on its placeholders only
because the Browser pane was hidden: TanStack Query pauses a retry while the page is not visible
(`focusManager` in `retryer.js`), and resumes it when the page is seen (G-36, closed). The card's
message was the browser's raw "Failed to fetch", now plain words (B-31).

## U-12 — The account menu, moved to the primary sidebar, renders and works in the browser

**From:** account-menu relocation, 2026-09-14 · **Who can run it:** owner (his own dev server is
already running); agent on a later turn once no other session holds the `apps/web` dev-server lock
**How:** Look at the primary sidebar (Chat/Machines/Settings) on any page. Passing: the account icon
sits in its footer, opens the same dropdown (email, Change password, Sign out, Sign out everywhere)
as before, and the chat header's Conversations panel no longer has it — check both expanded and
icon-collapsed sidebar states.
**Status:** verified 2026-09-14 — on production with the testing account, at 1440×900 (the Browser pane
reports 0×0 when hidden, which renders the sidebar as the mobile sheet). Exactly one Account button,
inside the sidebar footer and none outside the sidebar, so the Conversations header no longer has it.
Expanded, it opened agent@sparstrow.com with Change password, Sign out and Sign out everywhere. Collapsed
with Ctrl+B (the sidebar has no trigger button), the button stayed in the footer at 28×28 and opened the
same three items; Ctrl+B expanded it again. Nothing in the menu was clicked, and the console showed no
errors after a marker and a reload.

## U-13 — An installed computer moves off the signing key and then updates from a release GitHub built

**From:** D-035, 2026-09-14 · **Who can run it:** agent on the owner's PC (it needs published releases
and the installed copy)
**How:** With 0.2.1 installed and connected, publish 0.2.2 through the workflow and add its `.sig`
([runbook](runbooks/daemon-release.md)). Passing: the log shows 0.2.1 checking, handing over and the
updater's `updated` 0.2.1 → 0.2.2, then `connected` from 0.2.2. Then publish 0.2.3 through the workflow
alone, with no `.sig` on the release. Passing: 0.2.2 installs it the same way, and the installed
executable's SHA-256 is the one on the 0.2.3 release.
**Status:** verified 2026-09-14 — second half: `daemon-v0.2.3`, built by workflow run 34848559699 with
no `.sig` on the release, was installed by 0.2.2's first check. It handed over at 09:22:21, the new
copy connected at 09:22:22, and the updater logged `updated` 0.2.2 → 0.2.3 at 09:22:27. The installed
executable's SHA-256 is 0.2.3's (`ac302635…480f`), one copy runs, no failure was recorded. First half: `daemon-v0.2.2`, built by workflow run 34848075138
(every step passed), got its `.sig` from the old key over byte-identical manifest bytes (SHA-256
`490a99a8…d6cb`; the key's public half is the one 0.2.1 trusts). On the owner's PC 0.2.1 handed over at
09:19:18, the new copy connected at 09:19:19, and the updater logged `updated` 0.2.1 → 0.2.2 at
09:19:24. The installed executable's SHA-256 is 0.2.2's (`12037d34…be77`), one copy runs, no failure
was recorded.

## U-14 — A second account cannot see the first account's work or send work to its computer

**From:** phase 1 exit gate (docs/runbooks/release-workflow.md), D-031, 2026-09-14 · **Who can run it:**
agent, with the testing account
**How:** While the owner's computer is online on his account, sign in as another account. Passing: its
Machines list and conversations contain nothing of his, and sending a message with no computer of its
own is refused rather than delivered to his.
**Status:** verified 2026-09-14 — on production with agent@sparstrow.com while the owner's PC was
connected on his account (its log shows `connected` at 22:42:21). The testing account's `/api/machines`
was empty before any test computer was paired, and afterwards listed only its own test computers.
`/api/conversations` returned only its 3 test conversations, all in scratch folders on this PC. With
its test computers disconnected, sending a message returned 503 "your machine is unreachable, so
nothing new can be sent" instead of reaching the owner's computer. The isolation tests
(`internal/api/isolation_test.go`) cover the same boundary for every endpoint.

## U-18 — A candidate is published, tested and promoted without being rebuilt

**From:** D-037 release channels, 2026-09-17 · **Who can run it:** agent for the first two steps,
**owner for the promotion** — publishing a stable artifact is his gate, and the first candidate
publication needs his go-ahead because it puts a public prerelease on the repository
**How:**
1. Tag `daemon-candidate-v<x.y.z>` on main. Passing: the workflow publishes a prerelease that is NOT
   marked latest, and `releases/download/daemon-candidate/sparstrowgen-update.json` names that
   release's installer. `releases/latest/...` is unchanged, so no installed computer sees anything.
2. A test computer with `SPARSTROWGEN_CHANNEL=candidate` (or installed with `-channel candidate`)
   finds it in **Check now** and updates; a computer on stable does not offer it.
3. The owner runs **Promote a daemon candidate**. Passing: `daemon-v<x.y.z>` appears as latest with
   the SAME SHA-256 as the candidate, its manifest names the new URL, and a stable computer updates
   to it and stays on the stable channel afterwards.
**Status:** step 1 verified 2026-09-17 — `daemon-candidate-v0.3.0` was published by the workflow as
a **Pre-release**, the moving `daemon-candidate` release was created, and its manifest names that
prerelease's installer (`version 0.3.0`). `releases/latest/…` still serves `0.2.3`, so no installed
computer sees anything. Steps 2 and 3 are open: a candidate computer updating needs a daemon running
from an installed location, which on this machine means the owner's own installed copy or a clean
Windows account (the U-7 and U-13 constraint), and the promotion is the owner's gate.

## U-16 — Two worktrees' stacks run side by side without touching each other

**From:** D-036 local isolation, 2026-09-17 · **Who can run it:** agent — needs Docker Desktop
running, which it was not when this shipped
**How:** In two checkouts, run `scripts\dev.ps1` (or `make db && make server`). Passing: two Postgres
containers under different Compose projects on different host ports, two servers, two web apps, two
daemon homes; `docker volume ls` shows a volume per project; stopping one leaves the other running;
and the main checkout is still on 5433/8080/3000 with the database it already had.
**Status:** verified 2026-09-17 — two Postgres containers ran at once under different Compose
projects, `sparstrowgen-1-db-1` on 5434 and a second on 5433, each with its own volume and the
owner's existing `sparstrowgen_db-data` untouched. `scripts\dev.ps1` then brought this worktree's
whole stack up on its own ports: migrations applied to the 5434 database, `/api/health` answered 200
on **8081**, nothing was listening on 8080, and the daemon used
`%LOCALAPPDATA%\sparstrowgen-dev-1`. It also found B-38.

## U-17 — The Makefile picks up the assigned stack

**From:** D-036 local isolation, 2026-09-17 · **Who can run it:** agent — needs `make`, which is on
neither this machine nor Git Bash, so it needs the Linux container (and therefore Docker)
**How:** `make stack` prints this worktree's ports; `make db` starts Postgres on the assigned port;
`make test-linux` runs against that port rather than 5433. Passing: each command uses the values in
`.stack.env`, and `make` regenerates that file when it is missing.
**Status:** verified 2026-09-17 — in `golang:1.27` with the worktree mounted, `make help` created
`.stack.env` through the `-include` rule, restarted itself and printed this worktree's own ports
(`make server` on :8081, `make web` on :3001). `make -n test-linux` showed
`TEST_DATABASE_URL=…host.docker.internal:5434/sparstrowgen_test…` — its own database, not 5433.

## U-15 — A tab with no live connection picks up an appearance change when it is looked at again

**From:** appearance (#29, #31), 2026-09-16 · **Who can run it:** owner, or an agent in a browser that
can hold real focus
**How:** Sign in, open two tabs — one on Machines (which holds no live connection), one on
Settings → Appearance. Change the accent in the second, then click back to the Machines tab. Passing:
it takes the new accent within a second of being looked at, without a reload.
**Status:** open — the same convergence was verified 2026-09-16 on production through the live
connection (a second tab on Settings → Updates followed a change within ~1.5 s, unprompted). The
focus path could not be run here: the Browser pane reports `document.visibilityState` "hidden" and
`hasFocus()` false even when a tab is fronted, and the refresh it triggers is exactly what a hidden
page suppresses (the same harness limit that produced G-36). The code path is
`refetchOnWindowFocus: "always"` on the session and appearance queries.

## U-19 — The signed-in screens show the monochrome default and the status colours

**From:** D-039 (appearance and status colour), 2026-09-19 · **Who can run it:** owner, or the agent once
he has signed the testing account in the Browser pane (CLAUDE.md §1)
**How:** After a deploy, or against a local stack, sign in and check: (1) a brand-new account opens
monochrome and Settings → Appearance lists Neutral first; (2) to (5) are the status screens, which D-040
changed to icon and word together and U-21 now covers; (6) an existing account that had Paper and Amber is now Mono and Neutral (migration 00015), and one
that chose anything else still shows what it chose. Passing: all six, in light and dark.
**Status:** open. What was seen 2026-09-19: the compiled stylesheet of the dev server, in a signed-out
browser, with the default resolving to Mono and Neutral, all eight mode and surface combinations and
four accents resolving to the right `--primary`, `--ring` and status colours, and every status
utility present. The screens above need a signed-in session and a backend, which that run did not have.

## U-20 — The database-backed appearance tests pass with the new default

**From:** D-039, migrations 00014 and 00015, 2026-09-19 · **Who can run it:** agent, with the local stack running
**How:** With the local Postgres up, run the store and API tests: `go test ./internal/store
./internal/api -run Appearance -count=1 -v` from `server/`. Passing: none skipped, and
`TestTheSessionCarriesTheAppearanceSoTheFirstPaintIsRight` sees a new account on `DefaultAppearance`
(Mono, Neutral). Also that migrations 00014 and 00015 apply and roll back, and that 00015 moves only an
account on exactly Paper and Amber.
**Status:** verified 2026-09-19 — against a scratch database on a running local Postgres (its own
databases, dropped afterwards): all seven Postgres-backed appearance tests plus the database-free one
ran and passed, none skipped. Migrations went up to 15, down to 14 (column defaults back to Paper and
Amber) and up again to 15. On four sample accounts the 00015 statement moved only Paper and Amber and
left Paper and Violet, Slate and Amber, and Mono and Neutral alone, with the mode unchanged. Not run:
00015 against the owner's real production row, which happens when it deploys.

## U-21 — The signed-in screens show every status as a colour, an icon and a word

**From:** D-040 and B-41, 2026-09-19 · **Who can run it:** agent on production; some rows need the owner
**Status:** **partly verified 2026-09-20**, on `app.sparstrow.com` with a real connected computer
(the agent's own, paired that night) and again with it stopped. Icon classes and computed colours
were read from the DOM rather than judged by eye.

| Row | Result |
|---|---|
| (1) Machines list: online | **pass** — green tick and "Online" |
| (1) Machines list: offline | **pass** — `circle-minus` in `text-muted-foreground`, the grey dash, and "Offline" |
| (2) Profile: status under the name, and "Available" per agent | **pass** — green tick and "Available" for claude, codex and agy; "Offline" under the name when stopped, and "Waiting for this computer to connect and report its providers" |
| (2) Profile: amber triangle for a provider that is not installed, grey dash for one waiting | **not seen** — all three agents are installed on this computer, so neither state could be produced |
| (3) Add computer: blue arc, amber triangle, no icon on "Approve this computer?" | **not run, deliberately** — clicking it hands the pairing link to the owner's installed daemon, which is exactly what B-43 does. Safe to run once #52 is deployed |
| (4) Settings → Updates, the six states | **not reachable** — the test computer is a development build, so the page correctly shows "Development build" and automatic updates as unavailable, and none of the six appear |
| (5) Chat: amber notice above the composer | **pass** — `triangle-alert` in `text-warning`, with the composer and Send both disabled |
| (5) Chat: the strip stays quiet while the computer is asleep | **pass, after a fix** — it was rendering an empty bordered band (B-44) |
| (5) Chat: amber triangle and "Not installed" on a provider chip | **not seen** — needs an agent that is not installed |
| (6) A failed turn: red octagon and a red title | **not run** — needs a turn to fail, which spends the owner's real quota |
| (7) Folder picker: amber triangle on the "not inside a git repository" hint | **not run** |
| (8) Wrong password, resent confirmation | **not run** — this agent does not type into password fields |
| (9) Toasts carry their icons | **not run** |
| (10) Red words are readable | **pass, measured** — `--destructive-text` is 6.41:1 on Mono light and 6.04:1 on Soft light, its worst case, against every ground. The Disconnect button was seen in light and dark |
| Reduced motion stops the arc | **not run** |

**What is left** is mostly states that need a computer missing an agent, a release build, or a turn
that fails. The cheapest way to close several at once is a second test computer without all three
CLIs installed.

## U-22 — The light themes show the darker provider colours and stay recognisable

**From:** D-041 and B-39, 2026-09-19 · **Who can run it:** agent
**Status:** **verified 2026-09-20**, on `app.sparstrow.com` in light mode, against a conversation
carrying both a claude and a codex turn and a conversation list carrying all three provider marks.

Contrast was **measured, not judged**: each token was resolved through the live stylesheet, painted
to a canvas so `oklch()` became real sRGB, composited over its ground where the token is translucent,
and scored with the WCAG formula. (Two earlier attempts were wrong — reading `oklch()` strings as
RGB, then ignoring alpha — and were thrown away when `--foreground` on `--background` came back as
1.5:1 instead of 17.65:1.)

Worst case per token across Mono, Paper, Slate and Soft, on the page background, the pane and the
muted ground:

| Token | Worst ratio | |
|---|---|---|
| `--provider-claude` | 4.60 | pass |
| `--provider-codex` | 4.51 | pass |
| `--provider-agy` | 4.55 | pass |
| `--code-comment` | 4.54 | pass |
| `--code-type` | 4.55 | pass |
| `--capacity-out` | 4.52 | pass |
| `--success-text` | 6.28 | pass |
| `--warning-text` | 5.85 | pass |
| `--info-text` | 7.30 | pass |
| `--destructive-text` | 5.52 | pass |
| `--muted-foreground` | 5.43 | pass |
| `--foreground` | 15.20 | pass |

Everything clears 4.5:1 on every light surface, with Soft the tightest throughout. The three
providers sit at nearly the same lightness by design, so they are told apart by hue — claude warm,
agy blue, codex almost neutral — plus their own marks.

`--border` (1.05:1) and `--input` (1.14:1) are still faint, which is
[`KnownGaps.md`](KnownGaps.md) G-39 and not part of this check.

## U-23 — The chat surface works at phone width, and on a desktop inside the new shell

**From:** D-042 to D-045 and the shell wiring, 2026-09-19 · **Who can run it:** agent, on production
**Status:** **verified 2026-09-20**, on `app.sparstrow.com` with the owner's own session and his real
conversations, after #48 merged and deployed. Measured in the browser rather than eyeballed.

At 390 wide: the conversation list fills the screen (rows 358px, search 306px) with the tray at the
foot of the viewport and nothing opened for you; opening a conversation replaces the screen, the tray
is gone from the page entirely, the back arrow says "Back to conversations", the composer sits on the
bottom edge with nothing under it, and the header keeps the title and folder while dropping
Rendered/Raw and the token count. No horizontal overflow.

At 1440 wide: rail 60, pane 211, header 56 across the rest, carrying the title, the folder,
Rendered/Raw, "15.4k tokens" and the live status on one line; the first conversation opens by itself.
The browser's websocket opens in 143ms and stays open, so the status is reporting a live connection
rather than a stale guess.

**Found and fixed in the same pass:** the live status's short form read "None" on a phone, which says
nothing. It now reads "No computer".

This closes [`KnownGaps.md`](KnownGaps.md) G-21.

## U-24 — The design system artifact still describes the old shell

**From:** the shell wiring (#48), 2026-09-19 · **Who can run it:** agent
**Status:** open — **known drift, deliberately not fixed at 1am.** Recorded rather than rushed,
because this artifact is what every agent reads before writing UI, and a wrong one is worse than an
incomplete one.

`apps/web` gained `components/shell/` — `Rail`, `SectionTray`, `AppShell`, `AppHeader`, `PaneHeader`
and `LiveStatus` — and lost `components/ui/sidebar.tsx` and `components/product-sidebar.tsx`. The
artifact still carries a `SidebarNav` component whose README describes the shadcn sidebar, ending
"The `sidebar-*` tokens are aliases here; the live app has not defined them yet" — which is now
permanently true, because the component that would have used them is deleted.

**How the artifact is actually built** (confirmed 2026-09-20, so the next session does not have to
work it out):

- It is **spec-driven**, not hand-edited. The generators are `build.mjs`, `bundle.src.js`,
  `spec.txt`, `gen-tokens.mjs` and `icons.json` in the **session scratchpad**, not in the repo.
- `spec.txt` holds one `@@ Name|Group|height|subtitle` block per component, each with a `--readme`
  and a `--preview` section; `bundle.src.js` holds the React implementations and the export map that
  becomes `window.SparstrowgenDS`. `build.mjs` reads both, plus the provider mark paths straight out
  of `apps/web/components/chat/provider-icon.tsx`, and writes `ds/project/`.
- **The generators live only in a session scratchpad, so they may not survive.** If they are gone,
  they have to be rebuilt from the published `bundle.js` before anything can be added.

**The drift has grown since** (2026-09-21, first-run setup and the profile). Also missing, and in
the same register so that one pass closes all of it:

| Added to `apps/web` | What the artifact needs to say |
|---|---|
| `components/ui/avatar.tsx` | A person in a circle: the picture when there is one, initials otherwise, never an empty circle. Sizes 7 (rail, menu) and 16 (profile form). |
| `components/setup/setup-wizard.tsx` | The full-screen first-run surface, and its `Stepper` — hidden below two steps, because a progress indicator with no progress in it is decoration. |
| `components/setup/setup-card.tsx` | The compact rendering of the same steps, in the Chat pane. |
| `components/settings/profile-fields.tsx` | The one profile editor, used by both Settings → Account and the wizard. |
| `Composer` | Now takes `disabledTone`: amber for a fault, neutral for a step not taken (B-46). |

**No new tokens** were introduced by any of it, which is the part worth checking rather than
assuming — every surface above uses `background`, `muted`, `border`, `primary`, `success` and the
status colours that the artifact already documents.

**The step:** retire `SidebarNav`; add `Rail` (60px, hover overlay, the pin at 224px),
`SectionTray` (the phone's bottom navigation), `AppHeader` and `PaneHeader` (both 56px),
`LiveStatus`, and the five rows above; note the 768px breakpoint and the phone's list-then-detail
behaviour; update the component list and the "Live gaps" section in `project/README.md`; bump
`lastChange` in `project/design-system.json`.

## U-25 — Pressing Enter in the workspace name boxes

**Raised:** 2026-09-21, building workspaces
**Who can run it:** anyone with the app open

Both name boxes — first-run setup step two, and "Add a workspace" in Settings → Workspaces — are
ordinary forms with a single text input and a submit button, so Enter should create the workspace.
That could not be confirmed in the browser automation: its synthetic Enter does not trigger a form's
implicit submission. Confirmed as a limitation of the tool rather than of this change by trying the
same thing on the shipped registration form, which this change does not touch, and getting the same
nothing.

**The step:** open Settings → Workspaces, press "Add a workspace", type a name, press Enter. It
should be created without touching the button. Same on `/setup` step two for a new account.

**If it fails:** the forms need an explicit `onKeyDown` for Enter, which is three lines in
`components/workspaces/workspace-fields.tsx`.

## U-26 — Workspaces on production, and on an account with real history

**Raised:** 2026-09-21, building workspaces (D-050)
**Who can run it:** the agent, on `agent@sparstrow.com`; the owner, on his own account

Migration 00017 moves every existing conversation into a backfilled "Personal" workspace. That was
run against the local development database — one account, one conversation, moved correctly — and
against the test database by the suite. It has not run against production, where there are two
accounts and real transcripts.

**The step:** deploy, then check that every account has exactly one workspace called Personal, that
`SELECT count(*) FROM conversations WHERE workspace_id IS NULL` is impossible (the column is NOT
NULL, so a failed backfill would have failed the migration), and that signing in lands on the
existing conversations rather than on an empty Chat.

**If it fails:** the migration refuses rather than guessing — it raises if any conversation is left
without a workspace — so a failure is a refused deploy, not a silent loss.

## U-27 — Machine assignment with two computers actually online

**Raised:** 2026-09-22, building machine assignment (D-051)
**Who can run it:** the owner, with two computers connected; the agent, with two test daemons

Every branch was exercised: the Go suite pairs a real computer over a real websocket and asserts
that removing it from a workspace makes that workspace's send 503 while another workspace's
succeeds, that the provider list and folder browsing follow the same rule, and that a stop reaches
the computer running the turn. The browser checks were done against two computers that existed as
rows but were **offline**, because this machine has one daemon and CLAUDE.md forbids using the
owner's install for a test.

**The step:** with two computers connected at once, assign one to Personal and the other to Work,
then send a message in each and check it ran on the intended computer. G-44 is the reason this is
worth doing by hand: with both online, which one runs the turn is not yet visible.

**If it fails:** the routing is in `hub.Target` and is covered by
`TestAWorkspaceRunsOnlyOnItsOwnComputers`, so a failure here is most likely about *which of two
online* computers was picked, which is G-44 rather than a bug.

## U-28 — The working indicator with reduced motion turned on

**Raised:** 2026-09-23, building D-052
**Who can run it:** anyone who can turn on the OS setting (Windows: Settings → Accessibility →
Visual effects → Animation effects off)

The browser check confirmed the `prefers-reduced-motion` rules are in the served CSS. The Browser
pane cannot emulate that setting, so they were never seen taking effect.

**The step:** with animation effects off, send a message. The grid should sit still and dim, and
"working" should be plain grey. The timer should still count.

## U-29 — The design system's WorkingIndicator no longer matches the code

**Raised:** 2026-09-23, building D-052
**Who can run it:** agent

The artifact's `WorkingIndicator` takes `elapsed` in seconds, or times itself from mount, and a
`variant`. The code's takes `startedAt` in epoch ms and chooses the variant from the provider
(D-052 says why). The artifact should gain `startedAt`, and a note on which agent gets which grid,
once the owner has confirmed that mapping. This is the same class of drift as U-24.

## U-30 — Daemon candidate 0.3.1's own installer, before it is promoted

**Raised:** 2026-09-23, releasing B-52
**Who can run it:** the owner, or the agent on a computer put on the candidate channel

`daemon-candidate-v0.3.1` was built by the release workflow from `1b70627`. A test daemon built
from that same commit reported the live Claude list (Opus 5.5 and the rest) and ran an Opus 5.5
turn. The published `sparstrowgen-setup.exe` itself has not been installed anywhere.

**The step:** on a candidate-channel computer, install 0.3.1. Open the composer's model menu on a
claude conversation and check it lists Opus 5.5. Then promote 0.3.1 with **Promote a daemon
candidate** (Actions tab). Every computer with automatic updates on installs it within the hour.

## U-31 — A turn on Fable 5.1 with 1M context

**Raised:** 2026-09-23, B-54
**Who can run it:** the owner — it spends usage credits

The picker now offers Fable only as `claude-fable-5-1[1m]`, and the daemon passes that to `--model`
unchanged. That is the picker's own value, but no turn has been run on it here, because the CLI
says it "Requires usage credits".

**The step:** in a claude conversation, choose Fable 5.1 and send something short. The reply should
arrive labelled Fable 5.1. A failure naming the model means the `[1m]` tag needs to come off.

## U-32 — A real turn's exchange shows in Raw on production

**Raised:** 2026-09-23, D-053
**Who can run it:** agent, with the testing account and a test daemon from this change

**The step:** after the change deploys, run a test daemon built from it (CLAUDE.md, the agent's
testing account), send a short claude message, switch to codex and send another. In Raw, the latest
turn is open: the command, the catch-up in stdin, what claude reported loading and its usage, and
every line; Copy lines copies them. The first turn's line says how big its record is. An older turn
says "Not recorded".
**Status:** open

## U-33 — The owner's own computer records its turns once 0.3.3 is installed

**Raised:** 2026-09-23, D-053
**Who can run it:** the owner — promoting a candidate is his step (docs/runbooks/daemon-release.md)

Recording happens in the daemon, so until his computer runs a daemon from this change, his turns
show "Not recorded" in Raw. Nothing else changes for him.

**The step:** promote candidate 0.3.3 with **Promote a daemon candidate** (Actions tab). Once his
computer updates, send any message and open Raw: the new turn has a record.
**Status:** open

## U-34 — The design system has no entry for the raw exchange or the two-button switch

**Raised:** 2026-09-23, D-053
**Who can run it:** agent

CLAUDE.md asks for the design system artifact to stay true to `components/chat`. This change added
[`exchange-panel.tsx`](../apps/web/components/chat/exchange-panel.tsx) and made the chat header's
hand-built Rendered / Raw switch a shared
[`choice-toggle.tsx`](../apps/web/components/chat/choice-toggle.tsx), which the artifact lists as
"not in this system yet".

**The step:** once the owner has reviewed the feature, add ChoiceToggle to the artifact's
components and describe the raw exchange under Chat, or record why not.
**Status:** open
