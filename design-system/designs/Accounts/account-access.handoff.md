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
The prototype carries a simulated inbox so the email half of each journey can be clicked through.

## Component mapping

| Prototype element | Use | Notes |
|---|---|---|
| Column with product name, title, footer | existing `AuthShell` | Unchanged |
| Labelled input with hint | existing `Field` | Unchanged |
| Inline error | existing `FormError` | Now also carries an optional action ("Send the link again") beneath it |
| Submit with spinner | existing `SubmitButton` | Unchanged |
| "Forgot your password?", "Invited? Create your account", "Back to sign in" | **NEW — text link** | No link style exists in `components/ui`; `Button variant="link"` is the nearest, but these are muted, not primary |
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
| Six-digit code alongside the link (variant 3B only) | same token record | **no — not built**; only if the owner picks 3B |
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
- "Make it expire" in the inbox is scaffolding to reach the expired state. Do not port.
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
- Hint on the registration email field: "sparstrowgen is by invitation. Use the address your
  invitation was sent to."
- Not-invited error wording and the neutral "check your email" wording.
- Opening a link from an email ends whatever session this browser had and continues as the link's
  account.
- Seed data: the owner's address `srihari@sparstrow.com` was not looked up from production.

## Open questions

Each is rendered both ways in the review bar. The first option is shown by default; that is not a
decision.

1. **When an address isn’t invited.** *Say so* tells the person immediately, so a typo is caught
   on the spot, but anyone can probe which addresses are invited. *Same reply as everyone* reveals
   nothing, but a person who mistyped waits for an email that never comes.
   Recommendation: **Say so.** Invitation status is low-value to a stranger, and the neutral path
   costs a real invitee real confusion.
2. **When the password is chosen.** *After confirming email* stores nothing for an address until
   its owner proves it, and needs no "unconfirmed" sign-in error. *On the first form* is one step
   shorter but creates a half-account that can be signed into and refused.
   Recommendation: **After confirming email.** It removes a whole unhappy path the spec warns about
   ("no half-working account").
3. **How the email confirms.** *Link* is one click. *Link or code* also works when the email is
   read on a phone but registration was started on the computer.
   Recommendation: **Link only** for now; the invitation flow is usually started from the email
   itself, on the same device.
4. The post-success message is an inline notice in the prototype; the real app already has a toast
   (`sonner`). Either works; the notice stays visible until dismissed, which matters for the
   "other browsers were signed out" message.

## Not included

- Inviting or removing people from inside the product — out of scope in the spec.
- Connecting a computer after sign-up — US2, designed next.
- Changing password while signed in — already built (`change-password.tsx`), unchanged.
- The chat surface itself — stand-in only.
