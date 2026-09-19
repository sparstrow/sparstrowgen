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

**From:** D-040 and B-41, 2026-09-19 · **Who can run it:** owner, or the agent once he has signed the
testing account in the Browser pane (CLAUDE.md §1)
**How:** Sign in, in light and dark, and check each place carries all three: (1) Machines lists an online
computer with a green tick and "Online", an offline one with a grey dash and "Offline"; (2) a computer's
profile does the same under its name, shows a green tick and "Available" for an available agent and an
amber triangle with the reason for one that is not installed, and a grey dash for one that is waiting;
(3) Add computer shows a blue turning arc while it looks for the computer, then an amber triangle if it
has not answered, and no icon on "Approve this computer?"; (4) Settings → Updates shows, per computer,
a green tick for "latest version", a blue "i" for "available", an amber clock for "waiting", a blue arc while
installing, a red octagon for a failure, and a grey dash while offline; (5) in Chat, the provider strip
shows an amber triangle and "Not installed" for a missing agent and stays quiet while the computer is
asleep, and the notice above the composer is amber with a triangle; (6) a failed turn shows the red
octagon and a red title; (7) the folder picker's "not inside a git repository" hint has an amber
triangle; (8) a wrong password shows a red octagon and the message, and a resent confirmation shows a
green tick; (9) a success, info, warning and error toast each carry their icon; (10) red words are
readable, including the Disconnect button and a destructive menu item. Passing: all ten, and the arc
does not turn with the system's reduced-motion setting on.
**Status:** open. Seen 2026-09-19 in a signed-out dev server, on a scratch page rendering the real
components and tokens (deleted afterwards): all seven tones inline, as a badge and as a quiet sentence,
in light and dark, with the icon on the first line of a wrapped sentence; the provider strip with a
blocked and a waiting provider; the composer notice; and the form error. Type-checked and linted. Not
seen: any of the real screens above, the failed-turn and folder-picker notices, and reduced motion.
