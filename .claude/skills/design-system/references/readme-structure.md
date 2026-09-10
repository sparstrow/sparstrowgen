# README.md structure

`design-system/README.md` renders as the masthead above every card, so it is the
first thing anyone reads and the highest-leverage prose in the system. Cards show
*what* the decisions are; the README says *what they mean* and *how to consume
them*.

Write it for two audiences at once: a person orienting for the first time, and
an agent about to build a screen. Both need the same things, which is convenient.

## Sections, in order

### 1. Identity
Product name, one line on what it is, and — if the designs use a fictional
tenant — name it. ("Tenant used in designs: NorthDoor Mfg.") Reviewers otherwise
wonder whether the sample data is real.

### 2. How to consume this
The single most practically useful section, and the one most often missing.
Exactly how someone mounts the system:

```markdown
1. Link `styles.css` in your `<head>`.
2. Load <fonts> — see `tokens/typography.css`.
3. Put `data-theme="light"` on the root for light mode; omit for dark (default).
4. Use `var(--primary)` etc. for all colour values — never a hex literal.
```

If components mount via a global, show the exact line.

### 3. Sources
In mirror mode: which real files this documents. In greenfield: which design
files the system was extracted from. This is what lets the next person verify a
claim instead of trusting it.

### 4. Product context
What the app *is* — modules, primary screens, the shell layout, the navigation
pattern. An agent building a new screen needs this to make it feel native rather
than technically-correct-but-foreign. A table of module → key screens works well,
plus a sentence on the shell ("icon rail → submenu sidebar → topbar → content")
and any universal interaction pattern ("row click → drawer → open card → full
page tab").

### 5. Content fundamentals
The rules that make copy feel like one product. This section punches far above
its length:

- **Tone** — one line. "Direct, professional, no fluff."
- **Casing** — sentence case on UI labels, title case on module names, UPPER for
  section heads. Say which, explicitly.
- **Numbers** — tabular numerals for anything compared in a column.
- **Codes & IDs** — the actual format, with examples (`SO-10434`, `BD-3684-KO`).
- **Currency and dates** — the exact formats, with examples.
- **Emoji** — say whether they are used. Usually: no, status is colour + icon + text.
- **Counts** — how a count next to a title is rendered.

### 6. Visual foundations
Prose summary of what the cards show — themes, brand colour with its actual
value, the status palette as a table of semantic → hue → use, type, spacing,
radius table, shadows, hover/focus treatment, icon set and stroke weight,
animation keyframes.

Duplicating the cards a little here is fine and intentional: the cards are for
looking, this is for grepping and for pasting into an agent's context.

### 7. File index
A tree of every file with a one-line comment each. This is what lets someone
find the right card in a system with forty of them, and what an agent reads to
know where a new file belongs.

## Keep it current

The README is prose and therefore the fastest thing to rot — `check` cannot
verify it. When a token changes or a component gains a variant, re-read the
README for a sentence that just became false. A README describing a system that
moved on is worse than none, because people believe it.
