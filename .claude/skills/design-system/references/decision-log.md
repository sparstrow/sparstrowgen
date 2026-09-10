# The decision log

`design-system/DECISIONS.md` records **why the design is the way it is** — the
reasoning behind each change, not the change itself.

This is a different document from `CHANGELOG.md`, and the split is deliberate:

| | Answers | Read when |
|---|---|---|
| `CHANGELOG.md` | *What* changed, newest first | Catching up on what's new |
| `DECISIONS.md` | *Why* it's like this | **Before changing a design choice** |

## Why this earns its keep

Design feedback almost always arrives attached to one screen. The owner looks at
a page or a prototype and says *"make these rows tighter"*, *"try it with more
colour"*, *"add something here."* There is a reason behind the request, and the
reason is worth considerably more than the change — because the change fixes one
screen, and the reason usually generalises to every screen like it.

Without a record, three things go wrong, all quietly:

- **The same debate recurs on every page.** Nobody remembers that row density
  was already settled, so it gets re-litigated per surface and lands slightly
  differently each time.
- **A later agent "cleans up" the deliberate thing.** Something that looks
  inconsistent is often the one place a real decision was applied. With no
  record it reads as an accident and gets normalised away.
- **Nobody can tell a decision from a default.** This is the expensive one. A
  rule nobody chose is obeyed exactly as faithfully as one that was carefully
  argued — which is how a generic starter doctrine can govern a whole app for
  months without anyone noticing.

## Entry format

Keep entries short enough that they actually get written. A log that feels like
paperwork stops being maintained, and a half-maintained log is worse than none
because it implies the unrecorded decisions were never made.

```markdown
## DD-007 — Machine rows use compact density

**Date:** 2026-08-17 · **Asked by:** owner · **Surface:** Machines

**Ask:** Tighten the rows; the list felt airy.

**Why:** "I need to see forty machines without scrolling." Density here serves
scanning, not taste — the page is a monitoring surface, and a full fleet on one
screen is the point of it.

**Generalises to:** Any list expected to exceed ~20 rows. Candidate rule for the
doctrine's spacing section, rather than a per-page override.

**Supersedes:** The comfortable density picked on the character board, which was
chosen before anyone had seen a realistic row count.

**Status:** applied to prototype · not yet in `DESIGN.md`
```

**`Why` and `Generalises to` are the two fields that matter.** Everything else is
bookkeeping. If you're writing an entry in a hurry, write those two properly and
leave the rest thin.

`Generalises to` is what turns one screen's feedback into a system rule. Fill it
in honestly — including `Generalises to: nothing, this surface only`, which is a
real and useful answer. It stops a local exception being promoted into a rule
that quietly reshapes screens nobody was looking at.

`Status` matters more than it looks, because a decision applied only to a
prototype is lost the moment the real page gets built by someone reading the
doctrine instead. Track the promotion explicitly:

- `applied to prototype · not yet in DESIGN.md`
- `promoted to DESIGN.md §5 — 2026-08-20`
- `superseded by DD-012`
- `rejected — kept for the record`

**Never delete an entry, including rejected ones.** A rejected decision is the
cheapest possible answer to "why don't we just…", and re-deriving why an idea
was dropped costs far more than the two lines it takes to keep.

## When to write one

- The owner reacts to a prototype or a built page and asks for something
  different — **this is the main case**, and the request usually arrives with
  its reason attached in the same sentence. Capture it while it's said; nobody
  reconstructs it later.
- A design choice was made where a reasonable person would have chosen
  otherwise.
- Something was deliberately *not* done, and its absence looks like an
  oversight.
- The doctrine itself changes.

Not every tweak needs an entry. The test: **would someone six months from now
reasonably try to undo this, or ask why it's like this?** If yes, log it. If
it's an obvious fix with no alternative worth considering, don't.

## Promoting a decision into the doctrine

When `Generalises to` names a real rule, the decision shouldn't stay in the log:

1. Add the rule to `DESIGN.md` — phrased checkably, per the `design-brief`
   skill's template.
2. Update the entry's `Status` to say where it landed and when.
3. Rebuild the design system so the cards reflect it.

The log is where a rule is *discovered*; the doctrine is where it becomes
binding. A decision that stays in the log forever is one that never actually
changed how anything gets built.
