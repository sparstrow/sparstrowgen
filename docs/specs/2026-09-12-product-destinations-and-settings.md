# Spec: Reach Chat, workspaces, Runtimes and Settings

| | |
|---|---|
| **Status** | **Draft — needs your correction and approval** |
| **Created** | 2026-09-12 |
| **Trigger** | "right now we have chat, we need settings ... we need a runtime [area]" and multiple workspaces must be created and managed |
| **Design** | not designed yet |
| **Open questions** | L-18 — which new account-wide customizations are useful enough to ship first |

> You suggested a sidebar. I translated that into the jobs you need to reach and deliberately left
> the navigation form open for the design step. This draft also refuses to invent a collection of
> switches merely so Settings looks populated.

## What's wrong today

The product has one obvious place: the conversation. That is enough while the computer is manually
started and there is almost nothing to configure. It stops being enough when installation,
pairing, machine access, provider status and updates become real product responsibilities.

Today, "Your machine is unreachable" is reported where the conversation happens, but there is no
clear place to understand or repair the machine. Account actions exist apart from that problem,
and adding more actions around the conversation would mix three different jobs: doing work,
managing the computers that perform it, and configuring the account or product.

## What I want instead

Chat, workspace management, computer management and account-wide configuration are distinct jobs I
can reach directly. I can step away from a conversation to switch workspaces, inspect or fix
something, and return without losing where I was. A setting lives with the thing it affects, so I
am not asked to understand an arbitrary collection of controls.

## User stories

### US1 — Move between work and administration without losing context (P1)

**As** a person using sparstrowgen **I want** to move between conversations, workspaces, computers
and settings **so that** switching, fixing or configuring the product does not interrupt the work
I was doing.

**Acceptance**

- **Given** I am reading or writing in a conversation, **when** I inspect a computer or an account
  setting and return, **then** the same conversation and my place in it are preserved.
- **Given** my computer is unreachable, incompatible or needs attention, **when** I follow the
  problem from Chat, **then** I reach the information or action for that computer rather than a
  generic dead end.
- **Given** I enter the product directly through a saved address or refresh outside Chat, **when**
  my session is valid, **then** I remain in the part of the product I intended to use.
- **Given** I am not signed in, **when** I try to reach any of these areas, **then** the account gate
  is consistent and I return to the intended place after signing in.
- **Given** one area cannot load, **when** I move to another, **then** the failure stays scoped to
  the affected job; a computer-status failure does not make saved conversations unreadable.
- **Given** I switch to another workspace and later return, **when** I reopen my earlier
  conversation, **then** I return to the correct workspace and conversation rather than a similarly
  named item from somewhere else.

### US2 — Put each control with the thing it affects (P1)

**As** a person configuring sparstrowgen **I want** account-wide choices separated from one
computer's controls **so that** I understand the scope and consequence before changing anything.

**Acceptance**

- **Given** an action affects one paired computer, **when** I look for it, **then** I find it with
  that computer and can tell that other computers are unaffected.
- **Given** an action affects my account or every place I use the product, **when** I look for it,
  **then** it has one account-wide home rather than being repeated inside conversations or
  computers.
- **Given** I change my password or end signed-in sessions, **when** the action completes, **then**
  its security consequence is explained and I am not sent to Coolify or another administrative
  system.
- **Given** a future preference can be restored to its normal behaviour, **when** I have changed it,
  **then** I can understand what the normal behaviour is and return to it.
- **Given** there are no useful new preferences yet, **when** this feature is designed, **then** it
  does not invent meaningless configuration simply to fill space; the existing account actions
  remain useful and machine actions stay with Runtimes.

## Edge cases

- **The current computer is revoked while its status is being viewed.** Its access and displayed
  state converge without making it look connected after revocation.
- **The current browser is signed out by an account action.** Unsaved conversation text is not
  silently sent, and the reason for returning to sign-in is understandable.
- **A destination is empty.** No conversations, no paired computers and no optional preferences are
  different situations and each tells the person what can meaningfully happen next.
- **A destination is slow or unavailable.** Existing information is not presented as freshly
  confirmed, and retrying does not duplicate a pairing or destructive account action.
- **The same product is open in two browser tabs.** A revocation, sign-out or account change made in
  one does not leave the other confidently displaying access that no longer exists.

## Out of scope

- **The visual navigation pattern.** A sidebar is the current direction, but choosing its form,
  hierarchy, density and responsive behaviour belongs to rendered design options.
- **A general-purpose administration console.** Coolify, databases, deployment secrets and release
  publishing remain outside the installed user's product.
- **Inventing customization options.** Theme, notification, agent defaults and other preferences
  enter this spec only when there is a real user scenario for them; L-18 records that open scope.
- **The behaviour inside Runtimes.** Installation, pairing, status, revocation and updating are
  specified in [`2026-09-12-managed-runtimes.md`](2026-09-12-managed-runtimes.md).
- **The account and workspace onboarding journey.** Account creation, personal workspaces and the
  first computer setup are specified separately. Workspace invitations and roles remain separate
  work.

## What I need from you

Confirm that Chat, workspace management, Runtimes and Settings are the jobs you intended. Design
will decide whether workspace management is its own destination or part of switching context.
Then tell me the first two or three account-wide customizations you would personally change; that
answer closes L-18 without blocking onboarding or managed runtimes.
