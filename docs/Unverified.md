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
**Status:** open

## U-4 — Add computer on an already connected computer changes nothing

**From:** #12, 2026-09-13 · **Who can run it:** owner
**How:** On the connected PC, choose Add computer. Passing: it says "This computer is already
connected", Machines still lists exactly one entry for it, and it stays Online without reconnecting.
**Status:** open

## U-5 — A disconnected computer is refused and needs approval to come back

**From:** US2 pairing (#11), 2026-09-13 · **Who can run it:** owner (it ends his pairing until he
pairs again)
**How:** Open the computer in Machines and disconnect it. Passing: it leaves the list, its log shows
it was disconnected and stopped, `machine-credential` is gone from `%LOCALAPPDATA%\sparstrowgen`,
and only a new Add computer plus approval brings it back.
**Status:** open

## U-6 — The owner confirms Settings → Updates reads and behaves as he wants

**From:** US3 (#15), 2026-09-13 · **Who can run it:** owner
**How:** He chose the design by pointing at Multica's Updates page and approved building the rest
without confirming it in the app first. Open Settings → Updates, try Check now and the automatic
switch, and say what to change. Passing: no changes, or his changes recorded as feedback.
**Status:** open

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
**Status:** open

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
**Status:** open

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
**Status:** open — `tsc --noEmit` and eslint pass on the changed files; not seen in a browser because
another session's `next dev` held the directory lock on `apps/web` for the whole turn

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
