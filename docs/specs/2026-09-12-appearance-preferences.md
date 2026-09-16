# Spec: Choose how sparstrowgen looks

| | |
|---|---|
| **Status** | **Approved 2026-09-16** — the owner chose the whole spec as one delivery |
| **Created** | 2026-09-12 |
| **Trigger** | "for the theme and user wide setting refer to this old app's setting I have added the colour theme, dark and light mode etc" |
| **Design** | Settings → Appearance, built in the existing settings pattern at the owner's direction (2026-09-16) rather than through rendered directions: one more screen in an established pattern |
| **Open questions** | none; the old app establishes the choices and account-wide scope |

> The previous app at `D:\Sparstrow\Sparstrowgen` is a read-only product reference. It confirms the
> intended choices below, but its Settings layout and implementation are not inherited. This spec
> defines what the person can choose and what remains true afterwards; rendered directions will
> decide how those choices look and behave on screen.

## What's wrong today

The current web app always starts in its dark expression. A person cannot ask it to follow the
computer's appearance, choose a light expression, or restore the surface and accent choices from
the previous app. There is also no account-wide preference, so introducing an isolated switch in
one browser would make the same account feel inconsistent elsewhere.

## What I want instead

I can choose the appearance that helps me read and work comfortably. My choice belongs to my
account, follows me between signed-in browsers, and applies without changing the meaning of status
or code colours. A new account begins with the familiar Paper surface, Amber accent and System
mode, then remembers whatever I choose.

## User stories

### US1 — Choose light, dark or the computer's preference (P1)

**As** a person using sparstrowgen **I want** its expression to be Light, Dark or follow my system
**so that** it remains comfortable in my normal working environment without repeated adjustment.

**Acceptance**

- **Given** I have not chosen a mode, **when** I first use the product, **then** it follows the
  current light or dark preference of my system.
- **Given** System mode is selected, **when** the system preference changes while sparstrowgen is
  open or between visits, **then** the product follows it without losing my other appearance
  choices.
- **Given** I select Light or Dark, **when** the system preference changes, **then** my explicit
  choice remains in effect.
- **Given** I change the mode, **when** the choice is accepted, **then** I can see its effect without
  reloading or losing my current conversation and the choice remains after I return.
- **Given** I sign in to the same account in another supported browser, **when** my settings load,
  **then** the same saved mode applies there.

### US2 — Choose a surface character and accent (P1)

**As** a person who spends time reading and writing in sparstrowgen **I want** to choose its neutral
surface and accent **so that** the product can feel warmer, cooler, softer or more neutral while
remaining recognisably the same product.

**Acceptance**

- **Given** I have not customised my appearance, **when** my account is created, **then** Paper is
  the surface and Amber is the accent.
- **Given** I choose a surface, **when** I select Paper, Slate, Soft or Mono, **then** the product's
  neutral background character changes consistently in both light and dark expressions.
- **Given** I choose an accent, **when** I select Amber, Violet, Blue, Teal or Rose, **then**
  interactive emphasis uses that accent consistently without recolouring success, warning,
  approval, danger or information meanings.
- **Given** I change either choice, **when** it is accepted, **then** I can see its effect without
  reloading and it does not interrupt or clear the work I am doing.
- **Given** I return later or sign in elsewhere, **when** my settings load, **then** the saved
  surface and accent apply with my saved mode.
- **Given** a saved choice is no longer understood by a newer product version, **when** it loads,
  **then** the product falls back to a readable supported choice rather than leaving the interface
  broken or invisible.

## Edge cases

- **The sign-in screen has no account preference yet.** It uses a readable product default and
  changes to the person's saved appearance after sign-in without an extended wrong-theme flash.
- **The account is open in more than one browser tab.** A saved appearance change converges across
  them; one tab does not permanently overwrite the account with stale values later.
- **Preference storage is temporarily unavailable.** The current page remains readable and the
  person is told the choice was not saved instead of being given false confirmation.
- **A mode or colour combination is viewed on a smaller screen or with browser zoom.** Content and
  controls remain readable and usable; customization does not lower the product's accessibility
  floor.

## Out of scope

- **Custom colour entry or an unlimited theme builder.** The first delivery restores the named,
  tested choices from the previous app.
- **Per-workspace or per-computer appearance.** Appearance is account-wide; a local override can be
  considered later only with a concrete conflicting user scenario.
- **Changing semantic status or code-syntax colours.** Those retain their meaning and distinction
  across every surface, accent and mode.
- **Choosing the Settings navigation or control layout.** That belongs to rendered design options
  after this behaviour is approved.

## What I need from you

Confirm that Paper with Amber following the system is the correct new-account default, and that
the named surface, accent and mode choices from the previous app are the complete first release.
Nothing designs or implements against this document until you explicitly approve it.
