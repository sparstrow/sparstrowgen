# Spec: Sign in, connect this computer and stay updated

| | |
|---|---|
| **Status** | **Approved 2026-09-12** |
| **Created** | 2026-09-12 |
| **Trigger** | "pairing, signing or creating an account without the secure code for the machine, and update, auto update setting is a must to build first; all other are later" |
| **Design** | not designed yet |
| **Open questions** | none |

## What's wrong today

A new person cannot create an ordinary account. The first and only account requires a temporary
code copied from a server log, which means using the product depends on help from whoever controls
the hosting. The computer connection has the same problem: it uses one shared secret and requires
the daemon to be started manually.

The hosted app can change independently from the daemon on the computer. There is no ordinary-user
way to check whether that installed part is current or to decide whether it updates automatically.

## What I want instead

I create my own account and sign in normally without a server code. The product helps me connect
the computer I am using without copying a shared machine secret. Once connected, I can leave safe
automatic updates on or turn them off and check deliberately. This is the smallest release that
makes the hosted app usable without understanding its hosting.

## User stories

### US1 — Create and use my own account (P1)

**As** a new person **I want** to create and sign in to my own account **so that** I can use the
product without asking the host for a secret from the server.

**Acceptance**

- **Given** I am new to sparstrowgen, **when** I register with my email address and password,
  **then** I can prove the email belongs to me and complete my account without a setup code,
  machine secret or access to Coolify.
- **Given** my account is ready, **when** I return, **then** I sign in with my own email address and
  password and no setup or pairing step becomes part of ordinary sign-in.
- **Given** the address is already registered, verification has expired or credentials are wrong,
  **when** I try to continue, **then** no duplicate or half-working account is created and I receive
  a safe explanation with the next available action.
- **Given** the existing production owner account already has conversations, **when** normal
  accounts replace first-owner setup, **then** that account and its existing work remain intact.

### US2 — Connect the computer I am using (P1)

**As** a signed-in person **I want** the product to find and connect this computer **so that** Chat
can use my local coding agents without me copying a shared credential or running a development
command.

**Acceptance**

- **Given** the local sparstrowgen component is already running, **when** I begin computer setup on
  that computer, **then** the product recognises it and lets me approve connecting it to my account
  without asking me to paste a secret or code.
- **Given** the local component is not installed or running, **when** setup looks for it, **then** I
  receive the correct Windows installation help and setup recognises it after it starts.
- **Given** the browser and local component do not belong to the same account or the connection
  cannot be proved, **when** pairing is attempted, **then** they do not connect silently and I am
  told how to retry safely.
- **Given** pairing succeeds, **when** the browser, product or computer restarts, **then** the
  computer remains connected to my account and becomes reachable without a terminal.
- **Given** I cannot connect a computer now, **when** I defer setup, **then** my account remains
  usable for reading existing work and I can resume computer setup later.

### US3 — Keep the installed component safely updated (P1)

**As** a person with a connected computer **I want** to check its version and control automatic
updates **so that** the hosted app remains compatible without interrupting agent work.

**Acceptance**

- **Given** I have not changed the update preference, **when** my computer is first connected,
  **then** safe automatic updating is on.
- **Given** I check for updates, **when** the check finishes, **then** I can distinguish current,
  update available, waiting for active work, updating and failed.
- **Given** automatic updating is on and no sparstrowgen agent work is active anywhere on that
  computer, **when** a trusted compatible update is ready, **then** it can update and reconnect
  while keeping the account connection.
- **Given** any agent work using that computer through sparstrowgen is active, **when** an automatic
  or manually requested update is ready, **then** the daemon is not stopped, replaced or restarted
  and the update waits until every active task has ended.
- **Given** the computer appeared idle earlier, **when** agent work begins before activation,
  **then** readiness is checked again and the update continues waiting rather than interrupting the
  new work.
- **Given** automatic updating is off, **when** an update is available, **then** the current version
  stays in use until I deliberately request the update.
- **Given** a download cannot be trusted or the new version cannot reconnect, **when** the attempt
  finishes, **then** the previous working version remains or is restored and the failure is
  understandable.

## Edge cases

- Registration, verification or pairing is closed midway and later resumed.
- The local component is installed while setup is already looking for it.
- The computer sleeps or loses its network during pairing or an update.
- Several conversations or browser tabs have agent work running on the same computer.
- An update remains waiting for a long-running task rather than treating elapsed time as permission
  to interrupt it.
- One browser tab changes the update preference while another is still open.

## Out of scope

- Multiple workspaces, workspace invitations and workspace management.
- A complete computer-management area, including renaming, provider/model inventory and revocation.
- Appearance themes and other account-wide customization.
- Release notes and release-publishing controls.
- macOS or Linux installation in the first delivery.
- The later `develop` to `staging` to `main` delivery workflow; it begins only after this release is
  implemented and verified on production.
