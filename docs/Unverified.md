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
**Status:** open — tried 2026-09-13; the page stayed on its loading placeholders instead (G-36)

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
