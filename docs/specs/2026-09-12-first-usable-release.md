# Spec: Sign in, connect this computer and stay updated

| | |
|---|---|
| **Status** | **Approved 2026-09-12 · revisions 1 and 2 approved 2026-09-12 · revision 3 approved 2026-09-13** |
| **Created** | 2026-09-12 |
| **Trigger** | "pairing, signing or creating an account without the secure code for the machine, and update, auto update setting is a must to build first; all other are later" |
| **Design** | US1: [`account-access` handoff](../../design-system/designs/Accounts/account-access.handoff.md) — built on the real backend 2026-09-12 · US2: machine-workspace shots awaiting direction choice · US3: not designed yet |
| **Open questions** | none |

> **Revision 2026-09-12.** Mapping this release against the phase 2 exit gate found five gaps. You
> said "go" to closing them; this is the wording. Each addition is marked *(added)*.
>
> 1. The exit gate asks for a useful message when a computer is too old; this spec did not.
> 2. Nothing said one account's work and computer are kept from another account.
> 3. Registration is by the owner's invitation, not open to anyone who finds the address.
> 4. A connected computer can be disconnected. The full computer-management area stays later.
> 5. A person who forgets their password can reset it through their email.
>
> I inferred two details: that allowing someone to register may, for this release, be a step the
> owner takes on the hosting side (a place in the product to invite people belongs with workspace
> invitations later), and what happens to agent work running on a computer at the moment it is
> disconnected (see Edge cases). Correct either if that is not what you mean.
>
> **Revision 2, 2026-09-12, from your review of the prototype.** Someone who is not invited is not
> turned away: their attempt becomes a request for access that reaches you, and they are told it
> was sent for approval. A separate place to invite people and approve requests is later work
> ([`2026-09-12-admin-invitations-and-approvals.md`](2026-09-12-admin-invitations-and-approvals.md)).
> Until it exists, I inferred that you learn of a request by email and approve it by allowing the
> address the same way you invite someone. You also chose: the password is set after the email is
> confirmed, and the email confirms with a link only.
>
> **Revision 3, 2026-09-13, from your review of US2 design shots.** Computers now have a dedicated
> product destination and a durable profile rather than living only inside Chat setup. This brings
> the account-scoped machines list into US2 so later approved computer capabilities have a stable
> home. The current design choice is whether list and profile use separate routes or master-detail.

## What's wrong today

A new person cannot create an ordinary account. The first and only account requires a temporary
code copied from a server log, which means using the product depends on help from whoever controls
the hosting. The computer connection has the same problem: it uses one shared secret and requires
the daemon to be started manually.

The hosted app can change independently from the daemon on the computer. There is no ordinary-user
way to check whether that installed part is current or to decide whether it updates automatically.

## What I want instead

The people I allow create their own accounts and sign in normally without a server code. Anyone
else can ask me for access, and I decide. Each
person's work and computers are theirs alone. The product gives those computers a durable place
where the person can connect one without copying a shared machine secret, inspect it, and
disconnect it again. Once connected,
they can leave safe automatic updates on or turn them off and check deliberately. This is the
smallest release that makes the hosted app usable without understanding its hosting.

## User stories

### US1 — Create and use my own account (P1)

**As** a person the owner has invited **I want** to create and sign in to my own account **so
that** I can use the product without asking the host for a secret from the server.

**Acceptance**

- **Given** I have been allowed to use sparstrowgen, **when** I register with my email address and
  password, **then** I can prove the email belongs to me and complete my account without a setup
  code, machine secret or access to the hosting.
- *(revision 2)* **Given** my email address has not been allowed, **when** I try to register,
  **then** no account is created, my request for access is sent to the owner, and I am told the
  request was sent for approval and that I can create my account with this address once it is
  approved, without learning whether any particular address already has an account.
- *(revision 2)* **Given** someone has asked for access, **when** their request is sent, **then**
  the owner learns of it without looking at the server, and once the owner approves it that address
  can create an account like any invited one.
- **Given** my account is ready, **when** I return, **then** I sign in with my own email address and
  password and no setup or pairing step becomes part of ordinary sign-in.
- **Given** the address is already registered, verification has expired or credentials are wrong,
  **when** I try to continue, **then** no duplicate or half-working account is created and I receive
  a safe explanation with the next available action.
- *(added)* **Given** I have forgotten my password, **when** I prove I control my registered email,
  **then** I choose a new password without asking the host, my other signed-in browsers are signed
  out, and asking for a reset never reveals whether an address is registered.
- *(added)* **Given** another person also has an account, **when** either of us signs in, **then**
  neither of us can see, search or open the other's conversations, or see the other's computers.
- **Given** the existing production owner account already has conversations, **when** normal
  accounts replace first-owner setup, **then** that account and its existing work remain intact and
  belong to that account.

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
- **Given** setup cannot tell whether the local component is present, **when** the check does not
  answer, **then** I am offered both retry and installation help rather than being told it is absent
  as a guess.
- **Given** the browser and local component do not belong to the same account or the connection
  cannot be proved, **when** pairing is attempted, **then** they do not connect silently and I am
  told how to retry safely.
- **Given** pairing succeeds, **when** the browser, product or computer restarts, **then** the
  computer remains connected to my account and becomes reachable without a terminal.
- *(added)* **Given** my computer is connected to my account, **when** another account is signed in,
  **then** it cannot send work to my computer or learn anything about it.
- *(added)* **Given** I have one or more connected computers, **when** I manage them, **then** I can
  see only my computers, open one to inspect its connection state, available providers and
  registered folders, and return later as more computer capabilities are added.
- *(added)* **Given** a computer should no longer act for me, **when** I disconnect it, **then** it is
  refused from its next attempt, what it still holds can never restore access, reconnecting it
  needs my approval again, and any other computer I have connected keeps working.
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
- *(added)* **Given** my computer's installed version is too old for the hosted app, **when** I try
  to use it, **then** I am told plainly that it needs updating and what to do, rather than seeing a
  generic offline or failed state, and my existing work stays readable.

## Edge cases

- Registration, verification, password reset or pairing is closed midway and later resumed.
- An invited address is typed with different capitalisation or surrounding spaces when registering.
- A password-reset request is made for an address that has no account; it looks identical to one
  that does.
- *(revision 2)* The same uninvited address asks again, or many times: the owner sees one request,
  not a flood, and repeated asking is slowed down like repeated sign-in attempts.
- *(revision 2)* An address is approved while its person still has the "request sent" message
  open; trying again then continues into account creation.
- The local component is installed while setup is already looking for it.
- The computer sleeps or loses its network during pairing or an update.
- Several conversations or browser tabs have agent work running on the same computer.
- An update remains waiting for a long-running task rather than treating elapsed time as permission
  to interrupt it.
- One browser tab changes the update preference while another is still open.
- *(added, inferred)* A computer is disconnected while agent work is running on it: that work ends,
  and the conversation records that it ended because the computer was disconnected, rather than
  carrying on with access that has been removed.
- *(added)* A computer is disconnected while it is asleep; when it wakes it is refused, not quietly
  reconnected.

## Out of scope

- Multiple workspaces, workspace invitations and workspace management.
- A separate administration place for inviting people, and approving or declining requests. Drafted
  in [`2026-09-12-admin-invitations-and-approvals.md`](2026-09-12-admin-invitations-and-approvals.md).
- Telling a person that their request was approved or declined.
- A complete computer-management area, including renaming and provider/model inventory.
  Disconnecting a computer is in scope; everything else about managing computers is later.
- Appearance themes and other account-wide customization.
- Release notes and release-publishing controls.
- macOS or Linux installation in the first delivery.
- The later `develop` to `staging` to `main` delivery workflow; it begins only after this release is
  implemented and verified on production.
