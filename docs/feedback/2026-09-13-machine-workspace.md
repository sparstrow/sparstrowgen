# Machine workspace — feedback round, 2026-09-13

Review of the interactive A + B machine-workspace prototype on branch
`codex/computer-pairing` (commit `fbe8d29`).

Captured live during the round, in his words. **Status: closed 2026-09-13 — item applied.**
Procedure: [`feedback-round`](../../.claude/skills/feedback-round/SKILL.md).

---

## 1 — Remove the lower profile details for now

> These are not required in the frontend right now

**Reading:** “These” refers to the selected lower region of the `FINANCE-LAPTOP` profile: the
**Registered folders** section and the **Connection** section containing Account, Activity and Last
seen. Remove those sections from the current frontend design; this does not say the underlying
machine data is unwanted later.

**Evidence supplied:** the selected browser region begins at “Registered folders” and ends after
the three “Connection” facts. Agent providers and the profile header are outside the selection.

---

## Triage

| # | Item | Destination |
|---|---|---|
| 1 | Remove Registered folders and Connection from the current profile | Specific change to the approved design; update prototype and design contract. Park the removed detail under `Later.md` L-19 |

## Outcomes

**1 — done.** Both selected sections were removed from online, offline and disconnected machine
profiles. Provider availability remains the final profile section; last seen remains in the header
and Machines list because it directly explains connection freshness. The mock no longer carries
folder paths, the prototype card and handoff match the change, and the US2 acceptance text no
longer promises registered folders in the current frontend.

**Durable preference:** future extensibility is a reason to give a machine a stable profile, not a
reason to fill that profile with information before a current workflow needs it. Recorded with the
selected direction in the machine-workspace shot README.
