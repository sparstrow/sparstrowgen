# Account access — handoff

| | |
|---|---|
| **Prototype** | `account-access.dc.html` (serve with the `design-system` launch config; opened as a bare file it has no styles or data) |
| **Provenance** | [`docs/specs/2026-09-12-first-usable-release.md`](../../../docs/specs/2026-09-12-first-usable-release.md), US1 |
| **Mode** | build |
| **Status** | draft — awaiting the owner's review |
| **Design system** | none yet. Tokens mirrored from `apps/web/app/globals.css` into `design-system/tokens/colors.css`; screen furniture mirrors `apps/web/components/auth/shell.tsx` |

## What this is

Every way into sparstrowgen for a person the owner has invited: create an account, confirm the
email, sign in, reset a forgotten password, and land in an account that shows only their own work.
Someone who was not invited is not refused: their attempt becomes an access request emailed to the
owner.
The prototype carries a simulated inbox so the email half of each journey can be clicked through.

## Component mapping

| Prototype element | Use | Notes |
|---|---|---|
| Column with product name, title, footer | existing `AuthShell` | Unchanged |
| Labelled input with hint | existing `Field` | Unchanged |
| Inline error | existing `FormError` | Unchanged |
| "Request sent" screen | existing `AuthShell` with two paragraphs and two links | No new component |
| Submit with spinner | existing `SubmitButton` | Unchanged |
| "Forgot your password?", "Create an account", "Back to sign in", "Try again" | **NEW — text link** | No link style exists in `components/ui`; `Button variant="link"` is the nearest, but these are muted, not primary |
| Read-only email on choose-password screens | **NEW — small read-only value** | Plain text under a label; not a disabled input |
| Resend with countdown | existing `Button` (link-styled) | Countdown text replaces the label |
| Server unreachable | existing `SignInUnreachable` | Unchanged copy |
| Checking session | existing `SignInChecking` | Unchanged |
| Account menu | existing `AccountMenu` | Stand-in; no change proposed |
| Notice after account creation / password change | **NEW — dismissible inline notice** | Could be the existing `sonner` toast instead; see Open questions |
| Chat sidebar and empty main | stand-in for `chat-surface.tsx` | Only to show whose work you land in. Not a design proposal |
| Review bar, simulated inbox, seed-account list | prototype scaffolding | **Do not port** |

## Token usage

| Needed | Exists? | Action |
|---|---|---|
| `--background`, `--foreground`, `--muted-foreground`, `--primary`, `--secondary`, `--input`, `--ring`, `--destructive`, `--border`, `--card`, `--popover`, `--accent`, `--radius` | yes, mirrored | — |
| `--provider-claude/codex/agy` | yes, mirrored | Only the stand-in list dots |
| Menu shadow | **no** | The prototype uses an inline `oklch(0 0 0 / 50%)` shadow on the stand-in menu only. The real menu already has its own; nothing to add |

## States

| State | Reachable in prototype | Notes |
|---|---|---|
| Populated | yes — "Owner signed in", `?state=populated` | The existing owner account with its nine conversations, intact |
| Empty | yes — "New account", `?state=empty` | A newly created account: "No conversations yet" |
| Loading | yes — "Checking session", `?state=loading`; every submit shows its busy label for 0.7 s | |
| Error | yes — "Server unreachable", `?state=error`; plus "email can’t send" and "too many attempts" toggles, and every inline validation error | |

## Data contract

Built 2026-09-12. Server routes in `server/internal/api/accounts.go`; decisions D-031 and D-032.

| Field / behaviour | Source | Status |
|---|---|---|
| Account email, password hash | `users` | **built** — any number of accounts; `users_only_one` dropped (migration 00009) |
| Who may register | `OWNER_EMAIL` + `ALLOWED_EMAILS`, normalised | **built** — configuration until L-18 |
| Confirmation link: 30 minutes, single use, newest wins | `email_links` (hashed token) | **built and tested** |
| Reset link, same rules | `email_links` | **built and tested** |
| Access request per uninvited address | `access_requests` | **built and tested** — owner emailed on the first request only |
| Sending the emails | SMTP over TLS; `MAIL_TRANSPORT=log` in development | **built; real delivery unproved (KnownGaps G-29)** |
| "Password changed" email | same | **built** |
| Signing out every other session on reset | `sessions`, in the reset transaction | **built and tested** |
| Conversations filtered to the signed-in account | `conversations.user_id`, hub per account | **built and tested** |
| Resend spacing of 30 s | in-memory per address and kind | **built and tested** — the browser countdown mirrors it |

`POST /api/auth/register {email}` → `{outcome: "check-email"|"request-sent", email}` ·
`POST /api/auth/register/resend {email}` · `POST /api/auth/links/check {kind, token}` →
`{usable, email}` or `{usable: false, reason, accountReady}` · `POST /api/auth/register/complete
{token, password}` and `POST /api/auth/password/reset {token, password}` → session cookie +
`{ok, email}` · `POST /api/auth/password/forgot {email}` → `{ok, email}` whatever the address.

Nothing streams. An unfinished registration is resumable purely from the email link.

## Interactions

- Registration, confirmation, sign-in, reset and sign-out are real state transitions over seed data.
  Behaviour to keep; every one needs a real endpoint.
- Email addresses are trimmed and compared case-insensitively. Keep.
- Registering an address that already has an account shows the same "check your email" screen and
  emails a "you already have an account" message instead of a link. Keep: it answers the spec's
  "without learning whether any particular address already has an account".
- A reset request for an address with no account shows the identical reply and sends nothing. Keep.
- Sending again invalidates the earlier link ("A newer link was sent"). Keep.
- An uninvited address shows "Request sent" and emails the owner once; asking again shows the same
  screen, counts the attempt and sends nothing. Keep.
- "Make it expire" in the inbox is scaffolding to reach the expired state. Do not port.
- "Approve" on the owner's request email is scaffolding standing in for allowing the address in
  hosting configuration. Do not port; the real approval place is L-18.
- "Sign out everywhere" in the stand-in menu behaves like "Sign out". The real one already works.

## Invented

Decided by the prototype, approved by nobody:

- Links last **30 minutes** and work **once**; only the **newest** link works.
- Resend cooldown of **30 seconds**.
- Sender shown as `sparstrowgen <agent@sparstrow.com>` — the same mailbox as `OWNER_EMAIL`, sending
  its own confirmation and reset mail; no separate mailbox was created for this.
- All email subject lines and body copy.
- A **"your password was changed"** email after every reset. Not in the spec; a common safety net.
- After choosing a new password you are **signed in straight away** in that browser, rather than
  being sent back to sign in.
- After confirming your email you land directly in the app with "Your account is ready."
- Sign-in footer rewritten to "Sessions last a week. Signing out everywhere ends them on every
  device." The current footer ("the machine that runs your agents") is no longer true with more than
  one account.
- The two links under the sign-in form and their wording.
- Hint on the registration email field: "If you were invited, use the address your invitation was
  sent to. Otherwise we’ll ask for approval."
- "Request sent" wording, the owner's request email, and "Try again" on that screen.
- A repeated request from the same address never emails the owner a second time.
- The sign-in link reads "Create an account" rather than naming invitations.
- Opening a link from an email ends whatever session this browser had and continues as the link's
  account.
- Seed data: the owner's address `srihari@sparstrow.com` was not looked up from production.

## Decided by the owner, 2026-09-12

Reasons in [`design-system/DECISIONS.md`](../../DECISIONS.md).

1. **An uninvited sign-up becomes an access request** (DD-001). Both "say it is not invited" and
   "same reply as everyone" were rendered and rejected.
2. **The password is chosen after the email is confirmed** (DD-002).
3. **The email confirms with a link only** (DD-003).

## Open questions

1. The post-success message is an inline notice in the prototype; the real app already has a toast
   (`sonner`). Either works; the notice stays visible until dismissed, which matters for the
   "other browsers were signed out" message.

## Not included

- Inviting or removing people from inside the product — out of scope in the spec.
- Connecting a computer after sign-up — US2, designed next.
- Changing password while signed in — already built (`change-password.tsx`), unchanged.
- The chat surface itself — stand-in only.

## Verification — 2026-09-12 (after folding the owner's decisions)

**Tested:** `http://localhost:4173/designs/Accounts/account-access.dc.html` (launch config
`design-system`), against this handoff's States and Interactions and US1's acceptance criteria as
revised. Scripted DOM clicks drove the real handlers; each item asserted the resulting screen,
error text, mail count, request list or conversation count. Full pass: **55 checks, 0 failures,
0 console errors.**

### Checklist
- [x] No variant controls remain; decisions shown in the review bar
- [x] Sign in: wrong password error; typing clears it and keeps the form; owner lands with 9 conversations
- [x] Account menu shows the signed-in email; Escape closes it; sign out returns to sign in
- [x] Register asks only for email; invalid email refused; invited address with spaces and capitals normalised
- [x] No code field; confirmation link opens choose-password; short password refused; new account lands empty
- [x] Reused confirmation link → "Your account is already set up" → Sign in prefilled
- [x] New person signs in again and still sees none of the owner's conversations
- [x] Already-registered address → same "check your email" screen and a "you already have an account" email
- [x] Forgot password: unknown and real addresses get identical copy; only the real one gets mail
- [x] Reset link → new password → signed in with the signed-out-elsewhere notice; "password changed" email; notice dismisses
- [x] Old password refused; reused reset link → "already been used"; expired link → "Send a new reset link" prefilled
- [x] Uninvited address → "Request sent"; owner emailed once; request listed as asked 1×
- [x] "Try again" keeps the address; asking again shows the same screen, no second owner email, asked 2×
- [x] Owner's request email shows pending 2×; Approve (scaffolding) marks it approved and clears the waiting list
- [x] After approval, trying again → "Check your email" → link → password → new empty account
- [x] Faults: email can't send (register, forgot); too many attempts (sign in, request — no request recorded); server unreachable; retry busy, stays down, recovers
- [x] Presets and `?state=`: loading holds, error sets the fault, populated clears it and shows 9, empty shows 0
- [x] Resend: 30 s countdown, enables, sends, restarts; the earlier link becomes "A newer link was sent"
- [x] Phone width 375 px: no horizontal overflow on sign in, check email or request sent
- [x] Screenshots checked visually: sign in, request sent with the owner's email open

### Found & fixed
- **Typing after an error deleted the submit button** (first round). The error-clearing selector
  `#form-error + .btn` matched the submit button. Fix: only the error element is removed.
- **Presets kept fault toggles on** (this round). "Server unreachable" set the fault and every later
  preset, "Start over" included, carried it forward, so the next walkthrough silently hit a dead
  server. Fix: presets reset all faults.
- **Autofocus console notice** from the native attribute colliding with programmatic focus;
  replaced with `data-autofocus`.
- **Invited-address rows right-aligned** in the seed panel; explicit class on the action cell.

### Found & not fixed
- Opening a confirmation or reset link while already signed in as another account continues as the
  link's account without asking. Recorded under Invented; the real behaviour is a backend decision.

### Environment caveats
- Screenshots time out while the Browser pane is hidden; every item was asserted structurally and
  two states were checked visually.
- Geist is not bundled; the prototype falls back to the system sans unless Geist is installed.

## Wired into apps/web — 2026-09-12

On mock data (`apps/web/lib/auth.mock.ts`, KnownGaps G-28). Routes: `/register`, `/forgot`,
`/verify?token=`, `/reset?token=`; sign in gained its two links and new footer. Mock seed and
magic addresses are documented at the top of the mock file.

**Tested:** `pnpm typecheck` and `pnpm lint` clean; `next dev` via the `web` launch config
(auto-assigned port), walked in the browser with scripted clicks on the real components.

- [x] Register: empty disables Continue; browser's own email validation blocks "not-an-email" (same as sign in); typing clears a server error
- [x] Uninvited → "Request sent" + one owner toast; Try again keeps the address; asking again counts 2, no second toast
- [x] `+maildown` and `+throttle` addresses show the server errors
- [x] Invited address normalised → "Check your email" + confirmation toast with Open link; resend counts down 30 s, enables, sends, shows "Sent again", restarts
- [x] "Use a different address" keeps the address; registering again supersedes the first link → "A newer link was sent"
- [x] Confirmation link → "Choose a password" with read-only email; short disables and marks invalid; valid shows busy label, lands on `/` with the mock-note toast
- [x] Reused confirmation link → "Your account is already set up" with only Sign in
- [x] Unknown token and missing token → "This link has expired"; `token=offline` → "Can’t reach the server", retry spins and stays
- [x] Already-registered address → same "Check your email", "you already have an account" toast
- [x] Forgot: unknown and real addresses get identical copy, only the real one gets a reset toast; throttle error clears on typing
- [x] Reset link → "Choose a new password", label and footer, busy label, lands on `/` with changed and signed-out toasts plus the "was changed" email toast
- [x] Reused reset link → "already been used"; a 31-minute-old link → "expired"; both offer "Send a new reset link"
- [x] Link superseded while its form is open → submit turns the page into "A newer link was sent"
- [x] Sign in (session response stubbed as signed out, see caveat): both links with correct hrefs, new footer, client navigation to `/forgot`, back, and to `/register`
- [x] Phone width 375 px: no horizontal overflow on `/register` or sign in; column 327 px
- [x] Screenshot checked: sign in with its links and a mock email toast

**Caveats.**
- The only console errors were CORS failures from `/` calling the API on :8080, which belongs to
  another checkout, serves a different build (its session reply omits `claimed`) and does not allow
  this dev port's origin. The new pages never call the API.
- Sign in was exercised with the browser's session request stubbed to "claimed, signed out". The
  local database has no accounts, and creating one would write to a database another session's
  server uses.
