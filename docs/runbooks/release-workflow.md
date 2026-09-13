# Intended product and release workflow

**Status: proposed for owner verification.** This is the destination, not a
description of automation that exists today. The branch workflow in phase 2
must not be activated until phase 1 has been completed and verified.

This runbook describes what the release experience should feel like from the
owner's and an installed user's side. The agent rules that enforce it live in
[`WORKFLOW.md`](../../WORKFLOW.md). The editable diagram is
[`release-workflow.excalidraw`](release-workflow.excalidraw).

```mermaid
flowchart LR
    P1["Phase 1 on the current main workflow<br/>Invited accounts · pairing · auto-start · safe updates"]
    V1{"Owner verifies the installed<br/>daemon end to end"}
    P2["Activate phase 2<br/>development → staging → production"]
    W["Isolated agent worktree<br/>feature branch + local stack"]
    D["develop<br/>shared integration"]
    S["staging<br/>web/API + candidate daemon"]
    U{"Owner approves<br/>the exact candidate"}
    M["main<br/>production release"]
    C["Coolify production<br/>app + API"]
    R["Stable signed daemon release"]
    I["Installed supervisor<br/>updates while idle or rolls back"]

    P1 --> V1 --> P2 --> W -->|PR + checks| D -->|promotion PR| S --> U
    U -->|approved| M
    U -->|changes needed| W
    M --> C
    M --> R --> I
```

## Phase 1 — make the daemon a managed product

Keep the existing `main`-only workflow while this phase is built. Feature
branches still reach `main` through pull requests, and `main` remains the
production deployment.

The owner approved the deliberately narrow
[`first usable release`](../specs/2026-09-12-first-usable-release.md): invitation-only accounts
isolated from one another, password reset, computer pairing and disconnecting, auto-start and safe
update controls. The broader account/workspace,
runtime-management, navigation and appearance documents remain later drafts. They do not expand
this phase merely because they are already written down.

The current static `DAEMON_TOKEN` remains valid while the replacement is
introduced. The server first learns both credential types, then paired daemons
are installed and verified, and only then may the shared token be retired. A
server release must never require a daemon version that users had no chance to
receive first.

From an installed user's point of view, the finished experience is:

1. Download one signed Windows installer from a stable product link. GitHub
   Releases may store the file, but release mechanics are not shown in the app.
2. The installer starts a per-user supervisor automatically when Windows signs
   in. It runs with the person's own files, `PATH` and coding-agent credentials;
   it is not a `LocalSystem` service.
3. Pair the computer in the browser. The server issues a separate credential
   for that machine and stores only a hash of it. No Coolify access and no
   shared production secret are given to the user.
4. The app confirms that this computer is connected and shows enough version and
   update state to check now, leave safe automatic updating on, or turn it off.
5. The supervisor checks the stable update channel and may prepare a download,
   but it does not switch versions or restart while any sparstrowgen agent work
   is active on that computer. It checks for active work again immediately
   before activation, verifies the signed download and checksum, switches
   versions, restarts and reconnects. If startup or compatibility checks fail,
   it rolls back.
6. Successful updates are quiet. The app surfaces only action that a person
   needs to take: pairing approval, an incompatible version, a failed update,
   missing provider authentication, or a disconnected computer.

This phase does not establish a permanent product navigation or a general
Settings area. Design covers only the first-use journey and the approved update
jobs. Workspaces, broader computer management, appearance and other preferences
follow later through their own approved designs.

### Compatibility during phase 1

Daemon and server releases use an expand/contract sequence:

1. Add protocol fields and server behaviour so old and new daemons both work.
2. Deploy the compatible server.
3. Release and update daemons.
4. Measure or visibly confirm that incompatible daemons are gone.
5. Remove the old behaviour in a later release.

Every daemon hello reports daemon version, protocol range and capabilities.
The server responds with compatibility and update status. Missing fields are
treated deliberately as an older client, never as a valid value guessed by the
server.

### Phase 1 exit gate

Phase 2 may begin only after the owner can verify all of these as a normal user:

- an invited person can create, verify and later sign in to an account without a
  server-log setup code, and an address that was not invited cannot;
- a person who forgot their password can reset it through their email;
- a second account cannot see the first account's conversations or send work to
  its computer;
- a clean Windows machine can install without cloning the repository;
- first setup can discover the local component, offer installation when absent,
  finish pairing when present, or be skipped and resumed later;
- browser pairing connects the computer without exposing or asking the person to
  paste a shared machine secret;
- the daemon starts after sign-in without a terminal;
- Chat can complete a real agent turn through the connected computer;
- the person can check update status and change the automatic-update preference,
  which begins enabled;
- a signed test update waits for idle work, updates, reconnects and preserves
  configuration;
- an update that was ready while idle remains pending if any agent starts before
  activation, and waits until every active agent on that computer has finished;
- an intentionally broken test update rolls back;
- an older daemon gets a useful compatibility message rather than failing
  mysteriously;
- a disconnected computer is refused on its next attempt and needs approval to
  reconnect, while another connected computer keeps working.

## Phase 2 — activate development, staging and production promotion

Once phase 1 passes its exit gate, create and protect the long-lived
`develop` and `staging` branches. The same codebase moves through three
environments; differences such as domains, database URLs and release channels
come from environment configuration, never branch-specific source changes.

| Lane | What it is for | Hosted target | Daemon channel |
|---|---|---|---|
| Feature branch in its own worktree | One agent's isolated change and local proof | Unique localhost ports, Compose project, database volume and credentials | Development build; never auto-updates |
| `develop` | Shared integration after feature checks pass | Integration checks; not the owner's production app | Development artifacts only |
| `staging` | The exact release candidate the owner tests | `app.staging.sparstrow.com` and `api.staging.sparstrow.com`, with separate staging Postgres | Candidate |
| `main` | The production version people use | `app.sparstrow.com` and `api.sparstrow.com`, with separate production Postgres | Stable |

### Normal promotion

1. Each coding agent works in its own worktree and feature branch. Its web,
   API, database, daemon, ports, token and Compose project name cannot collide
   with another agent's stack.
2. A pull request into `develop` must pass the repository checks and browser
   verification appropriate to the change. Nobody develops directly on
   `develop`.
3. A promotion pull request moves `develop` to `staging`. Build the web/API
   images and Windows daemon candidate from that exact commit and record their
   immutable digests or checksums.
4. Coolify deploys the staging application against the staging database. Test
   daemons follow the candidate channel. The owner tests the app through the
   staging domains as an ordinary user.
5. If changes are needed, they return through a feature branch and `develop`;
   staging is not patched by hand.
6. After owner approval, promote that exact candidate to `main`. Production
   must use the already-verified image digest and daemon artifact rather than
   rebuilding different bytes from the same source.
7. Verify production health and a real daemon reconnect. Only then publish the
   candidate in the stable channel so installed production supervisors update.

### Failure and rollback

- A failed feature or integration check stops before `develop`.
- A failed staging candidate is replaced by a newer candidate; it never reaches
  `main`.
- A failed web/API production release rolls back to the previous verified image
  digest. Database migrations must be compatible with that rollback.
- A failed daemon update rolls back locally to the previous signed version and
  reports the failure where the person checks update status.
- A production hotfix starts from `main`, is proved locally, and reaches
  production through an owner-approved pull request. It is then merged forward
  into `develop` so the lanes do not diverge.

## Owner verification before activation

When phase 1 is complete, review this workflow and confirm:

- the staging domains are the names you want;
- only you can approve `staging` → `main` and a stable daemon release;
- production never builds unverified bytes after staging approval;
- installed users never need GitHub, Git or Coolify access;
- normal daemon updates are silent, while failures and required action are
  visible;
- every agent has a genuinely isolated local stack before `develop` is opened
  to parallel work.

After that confirmation, change `WORKFLOW.md` from **Phase 1** to **Phase 2** in
the same pull request that creates the protected branches and deployment
automation. Until then, agents follow the current `main` pull-request path.
