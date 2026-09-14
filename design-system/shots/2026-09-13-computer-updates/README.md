# Computer updates (US3) — 2026-09-13

No images were generated this round. Three directions were offered as one-liners before spending:

| | Direction | Optimised for | Outcome |
|---|---|---|---|
| A | An Updates section on the machine profile | Keeping the profile calm | not chosen |
| B | Status-led: a badge on the Machines row and a banner on the profile | Noticing without opening anything | not chosen |
| C | Version and a small menu in the profile header | Minimal space | not chosen |

**Chosen instead:** a Settings area with its own menu, and Updates as a page in it, copying
Multica's desktop Settings → Updates. His words: *"I want a settings submenu for updates like
multica, I want the same desing"*, with a screenshot of Multica's page: a breadcrumb, the title
"Updates" and one line of description, then one card of rows — **Current version** (`v0.4.43`),
**Automatic background updates** with a switch, and **Check for updates** with a **Check now**
button.

## What this told us about taste

- Preferences live in **Settings, grouped in a settings menu**, not on the object's own profile. The
  machine profile stays limited to status, providers and disconnect (consistent with the
  machine-workspace follow-up).
- When he names a product he already uses, the reference is the design. Copy its structure and
  wording, adapting only what our product genuinely differs on.

## Verification — 2026-09-13

**Tested:** `http://localhost:3317/settings/updates` on the real backend (local API on :8317,
database migrated to 00011), with an uncommitted harness playing four paired computers over the real
daemon websocket — against his reference screenshot, spec US3 and `DESIGN.md`.

### Checklist

- [x] Settings menu: Account → Password and Computers → Updates, breadcrumb, active entry; the
  password page still works inside it
- [x] Populated: an online computer that updates itself (v0.2.0, switch, Check now); an offline one
  showing its last reported v0.1.2 with Check now disabled; an older daemon ("Not reported yet",
  switch unavailable, "This version cannot update itself", Download update → `/install`)
- [x] Check now with automatic updates on: waiting on 2 tasks → 1 task → "Installing v0.2.1" with
  Updating… → reconnected as v0.2.1, "You're on the latest version" — each step arriving live
- [x] Automatic updates off: toast, saved (the API returns `automaticUpdates: false`), switch off;
  Check now then says "v0.2.1 is available." with Update now, which waits, installs and ends current
- [x] Failed: "The download of v0.2.1 did not match its signed checksum…" in the destructive tone,
  with Try again
- [x] Empty: an account with no computers sees "No computers connected" and Add computer → `/machines`
- [x] Console clean on this page after the fixes below
- [ ] Error: not seen. With the API stopped the page stayed on its placeholders (G-36, U-11)
- [ ] Too old: not reachable locally without raising `MinDaemonProtocol`; the server side is
  `TestATooOldComputerIsToldToUpdateAndSentNoWork` (U-10)

### Found & fixed

- Base UI warned that the link-rendered Download update and Add computer buttons lacked
  `nativeButton={false}` — fixed.
- Chat's composer crashed reading `models.length` of null — root cause: the harness sent providers
  without a models list, which a real daemon never does. Fixed in the harness, not the product.

### Found & not fixed

- G-36: an unreachable API leaves Machines and Updates loading indefinitely. Predates US3.

### Environment caveats

- The browser pane was not drawing, so screenshots and pointer clicks failed. Buttons and switches
  were pressed with DOM `click()`, and every state was read from the page text.
- The harness gave the local account a session directly in the local database; no password was
  typed.
- Dark theme only; the light theme was not looked at.
