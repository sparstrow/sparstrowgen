# Spec: Invite people and approve access requests

| | |
|---|---|
| **Status** | **Draft — later scope; not part of the first usable release** |
| **Created** | 2026-09-12 |
| **Trigger** | "I need an admin app or login separately from my user app, where I can invite other emails, also if someone else signed up who I did not invited ... it should say the request has been sent for account approval. then in my admin side I ll approve it." |
| **Design** | not designed yet |
| **Open questions** | one, below |

> Drafted from your review of the account-access prototype. The first usable release already turns
> an uninvited sign-up into a request and tells you about it; this spec is the administration side
> that follows. "Separate from the user app" is kept as an outcome (ordinary use does not grant
> administration) rather than a decision about whether it is a different site, address or sign-in
> screen, which the design step decides.

## What's wrong today

After the first usable release, inviting someone and approving a request both mean changing the
hosting configuration, and requests arrive only as emails to the owner. There is no single place to
see who was invited, who joined, who is waiting, or to take any of that back.

## What I want instead

I administer sparstrowgen somewhere that ordinary use cannot reach. There I invite people by email,
see every request from someone I did not invite, and approve or decline each one. The people
involved hear the outcome without me writing to them myself.

## User stories

### US1 — Administer from a separate, protected place (P1)

**As** the owner **I want** administration kept apart from everyday use **so that** signing in to
use the product never, by itself, lets anyone invite people or approve requests.

**Acceptance**

- **Given** I administer sparstrowgen, **when** I go to administration, **then** I prove I am an
  administrator there, even if I am already signed in to use the product.
- **Given** an ordinary account, **when** its person tries to reach administration, **then** they
  are refused without learning what administration contains.
- **Given** I am signed in to administration, **when** I leave it idle or sign out, **then** that
  administrative access ends without signing me out of everyday use, and the reverse also holds.

### US2 — Invite people by email (P1)

**As** the owner **I want** to invite someone by their email address **so that** they can create an
account without me changing the hosting.

**Acceptance**

- **Given** an address with no account and no invitation, **when** I invite it, **then** that
  person receives an email that leads them to create their account.
- **Given** I have invited people, **when** I look at invitations, **then** I can tell who is still
  invited, who has created an account, and when each was invited.
- **Given** an invitation has not been used, **when** I withdraw it, **then** its email no longer
  leads to account creation and the address is treated as uninvited.
- **Given** an invitation email was lost, **when** I send it again, **then** only the newest
  invitation works.
- **Given** the address already has an account, **when** I try to invite it, **then** I am told so
  and nothing is sent.

### US3 — Approve or decline access requests (P1)

**As** the owner **I want** to decide on requests from people I did not invite **so that** I can say
yes to someone I did not anticipate without opening sign-up to everyone.

**Acceptance**

- **Given** people have asked for access, **when** I look at requests, **then** I see each
  address, when it first asked and how many times, newest first.
- **Given** a pending request, **when** I approve it, **then** that person receives an email that
  leads them to create their account, exactly as if I had invited them.
- **Given** a pending request, **when** I decline it, **then** no account can be created from it and
  the request leaves the pending list while remaining visible as declined.
- **Given** a declined address asks again, **when** its request arrives, **then** I can see it was
  declined before and decide again, rather than it silently being refused forever.
- **Given** there are no requests, **when** I look, **then** I can tell there is nothing waiting
  rather than that the list failed to load.

## Edge cases

- One address asks many times, or many addresses ask at once: the list stays usable and I am not
  emailed once per attempt.
- I approve a request for an address I have also invited: one path to account creation, not two.
- A request is approved in one browser tab while another still shows it as pending.
- The person creates their account with different capitalisation from the request.

## Out of scope

- More than one administrator, and roles between them.
- Removing or suspending an existing account.
- Workspace membership and invitations into a workspace.
- Approving computers or anything other than people's access.

## Open question

**Does a declined person hear that they were declined?** Telling them is clearer but can invite an
argument; staying silent leaves them waiting. My recommendation: tell them plainly, once, without a
reason.
