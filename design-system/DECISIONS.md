# Design decisions

Why a design is the way it is, recorded when the owner reacts to something rendered. Newest first.
Each entry names the surface, what changed, and the reason — the reason is the part that generalises
to the next screen.

---

## DD-004 — Prototypes mirror the live app's tokens until a design system exists

**2026-09-12 · all prototypes.** There is no `DESIGN.md` yet. Rather than invent a palette per
prototype, `tokens/colors.css` copies `apps/web/app/globals.css` verbatim. When `design-brief` and
`design-system` run, they take that file over.

## DD-003 — Email confirmation is a link only

**2026-09-12 · account access.** A six-digit code alongside the link was rendered and not chosen.
**Why:** people start from the invitation email on the same device, so a code adds a field, a wrong-
code error and a second thing to verify for a case that rarely happens.

## DD-002 — The password is chosen after the email is confirmed

**2026-09-12 · account access.** Asking for the password on the first form was rendered and not
chosen. **Why:** it creates an account nobody has proved they own, which can be signed into and
refused — the "half-working account" the spec rules out. Asking afterwards removes that path.

## DD-001 — An uninvited sign-up becomes a request, never a refusal

**2026-09-12 · account access.** Both "say the address is not invited" and "give everyone the same
neutral reply" were rendered; the owner rejected both. **Why:** he wants to be able to say yes to
people he did not anticipate. A refusal ends the conversation; a request keeps it open and puts the
decision with him. It must be honest: the request is recorded and reaches him, or the message would
be a lie.
