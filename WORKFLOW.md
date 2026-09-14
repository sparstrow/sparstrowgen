# Delivery workflow for agents

This file is the operational protocol for source branches, hosted environments
and daemon releases. It complements `AGENTS.md`; all engineering and design
rules there still apply.

## Active phase: Phase 1 — managed daemon first

The repository is deliberately still on the current delivery path:

```text
agent worktree → feature branch → pull request → main → production
```

Do not create or route work through long-lived `develop` or `staging` branches
yet. Do not configure staging auto-deployments yet. The owner will activate
Phase 2 only after the managed daemon exit gate in
[`docs/runbooks/release-workflow.md`](docs/runbooks/release-workflow.md) passes.

### Phase 1 agent rules

1. Implement the approved
   [`first usable release`](docs/specs/2026-09-12-first-usable-release.md): invitation-only account
   creation (an uninvited sign-up becomes an access request emailed to the owner), password reset
   and sign-in without the server-log setup code, with each account's work
   and computers isolated from every other account; guided installation, pairing and disconnecting
   without a shared machine secret; per-user auto-start; and safe updating with check-now,
   automatic-update and too-old-version handling. Account isolation ships before or with
   registration, never after (`docs/KnownGaps.md` G-27). Complete one story through the feature
   process in `AGENTS.md` before beginning the next. The broader account/workspace, runtime-management, navigation and appearance drafts are
   later scope and must not expand this release.
2. Preserve the existing `DAEMON_TOKEN` path while the paired path is being
   introduced. Removal is a later contract step after installed clients are
   proven.
3. Make protocol changes additive first. The server must accept the currently
   deployed daemon throughout the transition.
4. Test installer and updater behaviour with development or candidate artifacts.
   A source build must never replace itself automatically.
5. Do not expose GitHub Releases, Coolify or deployment credentials in the
   product UI. Installed users can check for updates and choose automatic-update
   behaviour, but do not interact with release plumbing. Release notes are later
   scope.
6. Do not implement a generic Settings page or commit to a sidebar layout. The
   first release designs only the approved update jobs; broader Settings and
   navigation remain later work.

### Gate to Phase 2

Agents must not change this file to Phase 2 based on code completion alone. The
owner must verify the approved first usable release against the phase 1 exit
checklist in the runbook as a normal installed user. Phase activation, branch
protection and deployment automation belong in one explicit owner-approved
change.

## Phase 2 protocol — inactive until the gate passes

When activated, replace the active-phase heading and current path above. Then
the rules in this section become mandatory.

```text
isolated feature worktree → PR to develop → promotion PR to staging
→ owner UAT of exact artifacts → owner-approved promotion to main
```

### Branch ownership

| Branch | Meaning | Direct work |
|---|---|---|
| `codex/*`, `feat/*`, `fix/*` | One bounded change in an isolated worktree | Yes |
| `develop` | Shared integration | Never |
| `staging` | Release candidate under owner testing | Never |
| `main` | Production source of truth | Never |

- Feature pull requests target `develop` and use squash merge unless the owner
  explicitly chooses otherwise.
- Only a promotion pull request moves `develop` to `staging`.
- Only the owner may approve `staging` to `main` or publication to the stable
  daemon channel.
- A rejected staging candidate returns through a feature branch and `develop`;
  nobody patches `staging` directly.
- A hotfix branches from `main`, reaches `main` through owner approval, and is
  merged forward to `develop` immediately afterwards.

### Local isolation

Each agent worktree owns a unique local environment. Never share a running
Compose project, Postgres volume, host port, daemon process, daemon credential
or browser origin between worktrees. An allocator or registry must assign these
values before Phase 2 is activated; manually reusing the repository defaults is
not isolation.

Development daemons use development builds and never auto-update. Tests must not
resolve or spend quota through a real agent CLI unless the repository's existing
explicit real-agent gate is enabled.

### Build once, promote exactly

The staging promotion builds immutable web/API images and daemon artifacts from
one commit. Record image digests plus daemon checksums and attestations. The main
promotion reuses those exact bytes; rebuilding after approval creates a different
candidate and is forbidden.

Configuration belongs to the environment:

- staging and production have separate Postgres resources, domains, credentials
  and machine registrations;
- staging uses the candidate daemon channel;
- production uses the stable daemon channel;
- code must not hardcode a domain, database or release channel based on its Git
  branch.

### Compatibility and release order

Use expand/contract for every server-daemon protocol change:

1. server accepts old and new shapes;
2. compatible server deploys;
3. daemon candidate passes staging and becomes stable;
4. installed daemons update and reconnect;
5. old protocol support is removed only in a later release with evidence that
   the supported compatibility window permits it.

The installed supervisor may prepare an update in the background, but it must
not change versions or restart while any sparstrowgen agent work is active on
that computer. It checks again immediately before activation, verifies the
checksum, keeps the previous version, verifies restart and reconnect, and
rolls back on failure. A production server must never require an unpublished
daemon.

### Evidence and authority

- A green build is not staging approval. Record the checks actually run and the
  owner-visible candidate actually tested.
- A source commit is not an artifact identity. Record the immutable image digest
  and daemon checksum promoted.
- Agents may prepare promotion pull requests and evidence. They may not approve
  owner gates, push protected branches, publish stable artifacts or change
  production configuration without explicit authority.
- If automation behaves differently from this protocol, stop promotion, read
  the actual output and update either the implementation or this document. Do
  not silently invent a workaround.
