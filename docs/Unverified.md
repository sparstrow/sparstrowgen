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
