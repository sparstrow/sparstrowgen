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

The backend's brief for US1. Checked against [`Capabilities.md`](../../../docs/Capabilities.md).

| Field / behaviour | Source | Deliverable? |
|---|---|---|
| Account email, password hash, confirmed flag | users table (exists) | **partly** — accounts exist; the confirmed flag and removing the one-account index do not |
| Invitation list (address, trimmed and case-folded) | owner-maintained list | **no — not built.** Capabilities says hosting configuration is acceptable for this release |
| One-time confirmation link, 30-minute expiry, single use, newest-wins | new token record (hashed) | **no — not built** |
| Access request per uninvited address: address, first asked, times asked | new request record | **no — not built** |
| Owner emailed on an address's first request only | email delivery, to a configured owner address | **no — not built** |
| Approving a request | allowing the address like an invitation (hosting configuration) until L-18 | **partly** — same mechanism as invitations, once that exists |
| One-time reset link, same rules | new token record | **no — not built** |
| Sending the four emails | email delivery | **no — not built.** `agent@sparstrow.com` exists for end-to-end testing; the sending path is undecided |
| "Password changed" email | email delivery | **no — not built** (invented; see below) |
| Signing out every other session on reset | sessions table (exists) | **yes** — sign-out-everywhere already works |
| Conversations filtered to the signed-in account | conversations + hub | **no — G-27.** Must ship with registration |
| Throttle message "too many attempts — try again in 8s" | existing throttle | **yes** — same wording the server sends today |
| Resend cooldown of 30 s | server-enforced | **no — not built**; the countdown must mirror a server limit, not replace it |

Nothing streams. What must survive a refresh: an unfinished registration is resumable purely from
the email link, so no browser state is required between "check your email" and the link.

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
- Sender shown as `sparstrowgen <no-reply@sparstrow.com>`.
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
