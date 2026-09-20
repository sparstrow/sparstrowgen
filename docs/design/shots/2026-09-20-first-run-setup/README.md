# First-run computer setup — 2026-09-20

Four directions for the unbuilt half of the approved US2 in
[`2026-09-12-first-usable-release.md`](../../../specs/2026-09-12-first-usable-release.md):
discover the local component, offer installation when absent, finish pairing when present, or skip
and resume later. All four render the same moment — the computer has answered and is waiting for
approval — so only the idea varies.

| | Direction | Optimised for | Outcome |
|---|---|---|---|
| A | A dedicated full-screen route: numbered steps, nothing else on screen, skip at the bottom | Nobody gets lost; one thing at a time; room to grow | **chosen** — for a new account |
| B | Land in the real Chat; setup is a card in the conversation pane with the app alive behind it | Seeing the product immediately; setup never blocks | **chosen** — for resuming |
| C | No new screen: sign-in opens the existing Machines section with discovery already running | Costing almost nothing to build; resuming is the same place | rejected |
| D | The Bluetooth analogy: it searches, and this computer appears as a row in a list | Familiarity — everyone has paired a device | rejected |

## What the owner said

> "A is what we want when a new account is being setup, we would add lot more steps like adding
> workspace, account details etc.
>
> B is needed when there is some of the setup in the wizard is skipped or the app is being quit in
> between, we can continue from finishing from the app.
>
> We would like to have both."

So the two are not alternatives: **A is the guided path through first-run setup, B is how an
unfinished one is resumed from inside the app.** C and D were rejected implicitly — neither was
given a reason, so none is recorded. Both are single-purpose in a way A is not: C is the Machines
section doing its existing job, and D is a list, and a list is the wrong shape when only one
computer can ever appear in it (see below).

## What this told us about taste

- **A destination that will grow gets its own surface now.** The reason A won is not that it looked
  better; it is that workspace creation and account details are coming and need somewhere to live.
  The owner chose the shape for the feature he has not asked for yet. This is the same instinct as
  D-042 (a pane per section, because agents and schedules will fill it) — the second time it has
  appeared, so it is now a rule: *when a surface is the first of a series, build the series' shape.*
- **Skipping is expected, not exceptional.** B exists because he assumes people quit halfway. A
  setup flow here must be resumable by design rather than carrying a "don't lose your progress"
  warning.

## What the images could not settle, and the prototype has to

- **"Install" cannot be a step.** A's image shows step 1, Install, already complete with a check.
  Nothing checked it. The daemon dials out only, so there is no way to ask a computer whether the
  component is present without opening the `sparstrowgen://` link and waiting to see if anything
  claims the pairing (`docs/Capabilities.md`, the browser-recognising-the-component row). A tick
  there would be an assertion we cannot make. Install is a **branch inside Connect**, taken when
  nothing answers within eight seconds — which also leaves the numbered slots for the steps the
  owner actually wants to add.
- **D's second row was a promise the protocol cannot keep.** "Still searching…" implies more
  computers may arrive. Only one ever can: the one this browser is running on. Worth recording
  because the same mistake is available to any direction that renders discovery as a list.
- The four states, the eight-second unanswered moment, the refused-pairing case (G-40), and what
  any of this does at phone width. None of it is in an image.
