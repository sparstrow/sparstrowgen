# Spec: Install and manage a computer

| | |
|---|---|
| **Status** | **Draft — needs your correction and approval** |
| **Created** | 2026-09-12 |
| **Trigger** | "I want the auto update to happen ... The daemon should start automatically" and "pair, revoke access, check the models, check about the machine" |
| **Design** | not designed yet |
| **Open questions** | none; feasibility is checked after this draft is approved |

> Drafted from the hosting walkthrough and your description of using the product as an ordinary
> installed user. It deliberately says nothing about installers, background-service choices,
> release storage or screen layout; those decisions come after you confirm the behaviour.

## What's wrong today

The hosted app cannot do useful work unless a second program is manually started on the computer.
Today that means cloning the source, installing development tools, copying a shared secret, opening
a terminal and keeping a development command running. Closing the terminal or restarting Windows
makes the machine unreachable again.

The secret identifies every computer as the same machine. If it leaks or one computer should lose
access, the only repair is changing the secret everywhere. The app cannot explain which computer
is connected, which version it runs, whether it needs an update, or enough about its installed
agents to help when a model is missing.

The hosted app changes when production is deployed, but the program on the computer does not. A
new server can therefore meet an old computer with no safe way to update it or explain that the
two versions no longer understand each other.

## What I want instead

I install sparstrowgen on a computer once, pair it to my signed-in account, and then forget about
the machinery. It starts with me, stays compatible with the hosted app, and updates itself without
interrupting work. When I do need to look, I can understand each computer, what agents and models
it can use, and remove that computer's access without affecting any other one.

## User stories

### US1 — Install and pair a computer (P1)

**As** a person using the hosted app **I want** to install and pair my computer without cloning the
project or receiving hosting credentials **so that** the app can use my local coding agents without
making me a developer or server administrator.

**Acceptance**

- **Given** sparstrowgen has never been installed on my computer, **when** I follow the normal
  installation and pairing journey, **then** the computer becomes available to my signed-in
  account without Git, source code, Coolify access or a shared production secret.
- **Given** I am not already signed in, **when** pairing needs my approval, **then** I first prove
  which account the computer belongs to; finding a public pairing address is not enough to claim
  it.
- **Given** a pairing request is old, rejected, cancelled or already used, **when** it is submitted,
  **then** no computer receives access and I can start a fresh attempt with a clear explanation.
- **Given** pairing succeeds, **when** I close the setup journey, **then** the computer remains
  paired after the browser, app and computer restart.
- **Given** the computer cannot reach the hosted app, **when** I try to pair it, **then** I am told
  that connection failed and no half-paired computer is presented as ready.

### US2 — Understand and control my computers (P1)

**As** a person with one or more paired computers **I want** to see what each one can currently do
and revoke it independently **so that** "machine unreachable" becomes something I can understand
and access remains under my control.

**Acceptance**

- **Given** no computer is paired, **when** I try to use a local agent or inspect my computers,
  **then** I learn that a computer must be installed and paired and can begin that journey.
- **Given** a paired computer is connected, **when** I inspect it, **then** I can identify the
  computer and see that it is online, when it was last heard from, whether it is compatible, and
  whether it is updating or needs attention.
- **Given** its installed agents have been checked, **when** I inspect the computer, **then** I can
  see which agents are usable, which need attention, and which models each can genuinely offer.
- **Given** an agent cannot report its models or another detail, **when** I inspect it, **then** the
  missing information is stated honestly rather than filled with a guess.
- **Given** a paired computer is sleeping or disconnected, **when** I inspect it, **then** its last
  known information remains understandable but is clearly not presented as current.
- **Given** I revoke one computer, **when** it next connects or tries to perform work, **then** it is
  refused, its status reflects that, and every other paired computer continues to work.
- **Given** two computers have similar names, **when** I am about to revoke one, **then** I have
  enough trustworthy information to avoid removing the wrong computer.

### US3 — Start automatically and update safely (P1)

**As** a person who has already paired a computer **I want** its connection to start automatically
and remain compatible **so that** routine restarts and product updates do not turn the hosted app
back into a setup exercise.

**Acceptance**

- **Given** the computer has been paired, **when** I sign in to Windows, **then** it becomes
  reachable without opening a terminal or manually starting sparstrowgen.
- **Given** a compatible update is available and no agent work is running, **when** the computer
  checks for updates, **then** it updates, reconnects and keeps its pairing and configuration
  without asking me to understand where the release came from.
- **Given** agent work is running, **when** an ordinary update becomes available, **then** that work
  is not interrupted and the update waits.
- **Given** a download is incomplete, altered or cannot be trusted, **when** an update is attempted,
  **then** it is not installed and the currently working version remains available.
- **Given** an installed update cannot start, reconnect or work with the hosted app, **when** its
  safety check finishes, **then** the previous working version is restored and I can see that the
  update needs attention.
- **Given** the computer is too old for the hosted app, **when** I try to use it, **then** I see a
  useful compatibility explanation and recovery action instead of a generic offline state.
- **Given** an update succeeds normally, **when** I continue using the product, **then** release
  files, release pages and routine update notices do not become part of my work.

## Edge cases

- **The computer sleeps during pairing or updating.** Resuming must either continue safely or make
  the incomplete attempt clearly retryable; it must not create a second working identity by
  accident.
- **The network repeatedly connects and drops.** The computer can reconnect without asking for a
  new pairing each time, and an interrupted update never replaces a working version with a partial
  one.
- **Installed agents change while the computer is running.** Their availability can become current
  without reinstalling or pairing the computer again.
- **A provider is installed but signed out.** It remains visible with the action it needs; it is not
  mistaken for an absent provider or a broken computer.
- **A computer stays busy for a long time.** Ordinary updates keep waiting rather than killing the
  work. The person can still see that an update is pending.
- **The hosted app is temporarily unavailable.** Existing local configuration is preserved and the
  computer reconnects when service returns; repeated failure does not erase the pairing.
- **A revoked computer still has an old credential.** Possessing it never restores access and does
  not affect another computer's credential.

## Out of scope

- **Installing or signing into Claude, Codex or agy.** The computer reports what those agents need,
  but sparstrowgen does not own their accounts or installation journeys.
- **Remote control of the whole desktop.** Pairing authorises sparstrowgen's work, not screen or
  keyboard access.
- **Inviting multiple people into this deployment.** This spec replaces machine credentials; it
  does not change the current account model.
- **A visible release catalogue or release-management controls.** Installed users should not need
  to know that release storage exists.
- **Automatically updating development builds.** A developer running source needs predictable
  local code, not a background process that replaces it.
- **Forced ordinary updates that stop active work.** A separate emergency-security policy can be
  decided if a real incident makes that tradeoff necessary.
- **macOS and Linux installation in the first delivery.** The current user scenario is Windows;
  compatibility must remain possible, but supporting platforms nobody is using would delay the
  first usable installation.

## What I need from you

Read the scenarios as the person installing and using sparstrowgen. Correct anything that feels
wrong—especially what should happen when an update has been waiting behind long-running work.
Nothing designs or implements against this document until you explicitly approve it.
