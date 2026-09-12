# Spec: Create an account and begin a workspace

| | |
|---|---|
| **Status** | **Draft — later scope; account creation and initial pairing moved to the approved first usable release** |
| **Created** | 2026-09-12 |
| **Trigger** | "I want account to be created and then lets have multiple workspace creation and manage as well. Then the setup should take to computer setup." |
| **Design** | not designed yet |
| **Open questions** | none; the account-to-computer sequence below is inferred from the owner's scenario |

> I inferred that a workspace is a separate area of work inside an account, not a folder on the
> computer. I also inferred that a computer is paired once to the person and can then be made
> available where that person works, rather than being reinstalled for every workspace. Correct
> either inference if that is not what you mean.
>
> The smaller first delivery is approved in
> [`2026-09-12-first-usable-release.md`](2026-09-12-first-usable-release.md). This draft now retains
> the later workspace-management outcomes and must not expand that first delivery.

## What's wrong today

A fresh deployment does not offer normal account creation. The first person has to read a secret
code from the server's startup log and use it to create the only account. That proves ownership of
one private deployment, but it is not an experience another user can complete and it deliberately
prevents a second account.

After that account exists, the product opens directly into Chat. There is no separate place to
organise different bodies of work, and no guided path from "my account exists" to "my computer is
ready to run an agent." The person has to understand the daemon and hosting arrangement before the
product can do anything useful.

## What I want instead

Anyone I allow to use the product creates and verifies their own account without access to server
logs or Coolify. They begin in a workspace that keeps one body of work separate from another and
can create and manage more workspaces later. The first setup then helps them connect a computer,
but they can skip that part and return to it when they are ready.

## User stories

### US1 — Create and recover my own account (P1)

**As** a new user **I want** to create and verify my account through the product **so that** I do
not need a setup code or help from the person who hosts it.

**Acceptance**

- **Given** I have never used the product, **when** I register with an email address and password,
  **then** I prove that the email belongs to me before the account can access private work.
- **Given** I am creating an ordinary account, **when** I complete registration, **then** I am
  never asked for a code copied from server logs or for access to the hosting system.
- **Given** my account has already been verified, **when** I return and sign in normally, **then** I
  use my own account credentials and no setup or pairing code is part of sign-in.
- **Given** the email is already registered, **when** registration is attempted again, **then** no
  second identity is created and the response does not reveal more account information than is
  safe.
- **Given** verification is expired, already used or incorrect, **when** I try to finish account
  creation, **then** I can request a fresh attempt without creating a half-working account.
- **Given** I forget my password, **when** I prove control of the registered email, **then** I can
  choose a new password without asking the host to edit production data.
- **Given** an owner account already exists from the current deployment, **when** the new account
  model arrives, **then** that account and its conversations remain intact and it no longer has a
  special login journey.

### US2 — Create and manage separate workspaces (P1)

**As** a signed-in user **I want** more than one workspace **so that** unrelated work, computers
and conversations do not become one undifferentiated account.

**Acceptance**

- **Given** I have just created my account, **when** I begin using the product, **then** I create or
  confirm my first workspace before connecting a computer.
- **Given** I have several bodies of work, **when** I create another workspace, **then** its
  conversations and configuration begin separately from the existing workspace.
- **Given** I return later, **when** I choose a workspace, **then** I return to that workspace's own
  work without data from another workspace being presented as part of it.
- **Given** a workspace name no longer describes the work, **when** I rename it, **then** its
  conversations and computer access remain attached to the same workspace.
- **Given** I no longer need a workspace, **when** I manage it, **then** I can first put it aside
  without destroying it; permanent deletion is separate, clearly consequential and confirmed.
- **Given** one computer is available to me, **when** I work in more than one workspace, **then** I
  do not reinstall or re-pair that computer for each workspace, and I can understand where it is
  allowed to work.

### US3 — Complete or defer the first computer setup (P1)

**As** a user whose account and first workspace now exist **I want** setup to help connect my
computer **so that** I reach a usable product without already knowing what a daemon is.

**Acceptance**

- **Given** my account and first workspace are ready, **when** first setup continues, **then** the
  next outcome is connecting a computer rather than dropping me into a Chat that cannot send.
- **Given** the computer component is already running on this computer, **when** setup looks for it,
  **then** it is found and I can approve connecting this computer to my account.
- **Given** the computer component is not found, **when** setup checks this computer, **then** I get
  the correct installation package and plain instructions, and setup can recognise it after the
  installation completes.
- **Given** setup cannot tell whether the component is present, **when** the check fails, **then** it
  explains the problem and offers both retry and installation help rather than declaring the
  computer absent as a guess.
- **Given** I do not have the computer with me or do not want to connect it yet, **when** I skip
  computer setup, **then** my account and workspace remain complete and I can return to computer
  setup later from inside the product.
- **Given** I skipped computer setup, **when** I reach Chat, **then** existing work remains readable
  and the product explains why new agent work cannot run and where to connect a computer.
- **Given** the computer connects successfully, **when** setup finishes, **then** I can see which
  local agents are ready before I try to send the first real message.

## Edge cases

- **Registration is open but automated or abusive.** Proving an email is necessary but does not
  give one account access to another account or workspace.
- **The verification message is delayed.** The registration state remains understandable and can
  be resumed without starting over or creating duplicates.
- **A user belongs to no usable workspace.** They are helped to create one; another user's
  workspace is never used as a fallback.
- **Two workspaces have the same display name.** The person can still tell them apart before moving
  or deleting work.
- **The browser and local computer component are signed into different accounts.** They do not pair
  silently; the mismatch is explained and can be corrected.
- **Setup is refreshed or closed midway.** Completed account and workspace steps remain completed,
  while an incomplete computer connection resumes or restarts safely.
- **The installer requires a restart.** The setup can be resumed afterwards and recognises the
  installed component without requiring a new account or workspace.

## Out of scope

- **Inviting other people into the same workspace.** This delivery supports separate user accounts
  and multiple personal workspaces. Sharing, membership and roles need their own scenarios before
  they become part of workspace management.
- **Choosing the visual setup pattern.** The Bluetooth-device analogy and the supplied stepper are
  direction for design; the spec does not decide whether the final experience is a stepper,
  checklist, assistant or something else.
- **Automatically installing third-party coding agents.** Computer setup can explain their state,
  but those products retain their own installation and sign-in journeys.
- **Creating a workspace from a local folder automatically.** A product workspace and a project
  folder are different things; connecting folders belongs to agent work after a computer exists.
- **Hosting administration.** Users do not receive Coolify, database, DNS or deployment access.

## What I need from you

Confirm whether my workspace inference is right: it is a separate area inside your account, and a
computer is paired once to you rather than once per workspace. If that matches your intention,
approve this draft; otherwise correct that relationship before design begins.
