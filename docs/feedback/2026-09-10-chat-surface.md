# Chat surface — feedback round, 2026-09-10

First review of the chat surface built in `apps/web` (commit `36ece44`, direction A from the
[2026-09-09 shot round](../../design-system/shots/2026-09-09-chat-surface/README.md)).

Captured live during the round, in his words. **Status: closed 2026-09-10 — all six done.**
Procedure: [`feedback-round`](../../.claude/skills/feedback-round/SKILL.md).

---

## 1 — Model lists are invented, not real

> the models are not exactly the latest ones that are being sold. For example, AGI has Gemini 3.8
> GGU vs. 3.8, but here it's still showing AGI-1, AGI-1, so even the exact models to be listed

**Reading:** "AGI" is `agy`. The comparison is between what `agy` actually offers and the
`agy-1` / `agy-1-mini` in our mock, which I invented.

**Evidence supplied** — screenshot of `agy`'s own Switch Model screen, transcribed here because the
image does not survive into the repo:

```
Antigravity CLI 1.2.0 — Gemini 3.1 Pro (High)

Switch Model
  Gemini 3.8 Flash
  Gemini 3.7 Flash
  Gemini 3.6 Flash
> Gemini 3.1 Pro            (current)
  Claude Sonnet 4.6 (Thinking)
  Claude Opus 4.6 (Thinking)
  GPT-OSS 120B (Medium)

  Effort  ◀ ──low────────high──▶   "Deepest reasoning for complex problems"
  status bar: Gemini 3.1 Pro · high
```

**Consequences, raised in the round:**

- `agy` is a **multi-vendor router**, not a single-model provider. It serves Gemini, Claude *and*
  GPT models. The mental model "provider = vendor" is wrong, and the provider colour tokens
  currently imply it.
- `agy` carries an **effort dimension** (low→high, persisted in its status bar) that
  `Conversation.model: string` has nowhere to put.
- Model lists must be read from the CLIs, not typed from memory. `agy models` is already recorded
  in `blueprint.yaml`; the equivalent for `claude` and `codex` is not.

## 2 — Provider icons, and gemini removed

> I want you to use icons rather than possibly. For example, Claude, Codex, AGY, they all have their
> icons. Instead of just a dot, you can use icons too. Also, I don't want general to be part of this
> at all, since we are not going to use it. We need you to refer to multiple assets that have some
> accounts on these.

**Reading:** "rather than possibly" is dictation noise for *rather than dots*. "general" is
`gemini`. The last sentence is a genuine guess: most likely *only include providers he actually has
an account on* — which is why gemini goes. **Confirm at triage.**

**Consequence, raised in the round:** `gemini` is the only live example of the three-state
availability model (available / waitable / blocked, D-011). Removing it means "blocked" has no real
case left, so it either finds another one or stops being modelled.

## 3 — Geist

> I want you to Geist fonts

`layout.tsx` already loads `Geist` and `Geist_Mono` via `next/font/google`; unverified whether the
Tailwind token resolves to them or falls through to the default sans stack.

## 4 — Two dropdowns, not one merged menu

> For the dropdown on the models, I want a provider dropdown and then a model dropdown. First, I
> will choose which provider: codex or Claude, whatever I choose, then the next dropdown, which is
> models, needs to reflect accordingly.

Provider is chosen first; the model dropdown is scoped to that choice. Replaces the single grouped
menu currently in `composer.tsx`.

## 5 — Archive, offered at the point of deletion

> The conversation should also needs to be able to archived that option should be provided, when we
> try to delete the conversation, the pop up should also give option to archive there

**Consequence, raised in the round:** archive needs somewhere to go — a view or filter in the
sidebar — or it is a one-way disappear that is worse than deleting. Being reversible, it also wants
much lighter confirm copy than delete's.

## 6 — Search the conversation list

> the sidebar needs a search for conversations

Open at capture: titles only, or message text too. Titles alone is weak — several conversations are
literally "Untitled conversation". Full text is deliverable (the transcripts are in our own
Postgres); semantic search stays out of scope per the spec.

---

## Triage

| # | Item | Destination |
|---|---|---|
| 1 | Real model lists | Capability check first — the lists came from the CLIs, not memory. Then code. Findings → [`KnownGaps.md`](../KnownGaps.md) G-8, [`Decisions.md`](../Decisions.md) D-015, `Capabilities.md` |
| 2 | Provider icons | Code |
| 2 | Drop gemini | [`Decisions.md`](../Decisions.md) D-014, `Later.md` L-6 **deleted**, blueprint, `Capabilities.md` |
| 3 | Geist | Verify only — already correct |
| 4 | Two dropdowns | Code |
| 5 | Archive | Spec amendment (US4) + code |
| 6 | Search | Spec amendment (US4) + code. Scaling caveat → [`KnownGaps.md`](../KnownGaps.md) G-7 |

Nothing was refused, deferred, or needed the owner before starting. Item 2's ambiguous last
sentence is answered under its outcome below.

## Outcomes

**1 — done, and it found more than a wrong list.** `agy models` enumerates directly. `codex` has no
list command (names read from the shipped binary and its config, and the valid set turns out to be
*account*-dependent). `claude` has a `list_models` control request that costs nothing, but our 2.1.90
is too old to answer it — so its three aliases were resolved by running each once and reading the id
back out of `system.init`: `claude-opus-4-6`, `claude-sonnet-4-6`, `claude-haiku-4-5-20251001`.
Recorded as G-8.

**My first pass at this list was wrong too**, and in the way this round is about: I wrote
`claude-opus-5` / `claude-sonnet-5` from my own knowledge rather than from the machine, which does
not offer them. Two independent sources here — claude's own `system.init` and agy's roster — say
4.6.

Two things fell out of the real data and changed the design rather than just the content:

- A model needs an **id and a label** — `gemini-3.1-pro-high` next to "Gemini 3.1 Pro (High)".
  Neither is derivable from the other. `Model` is now a type, and a transcript stores the whole
  thing so an old turn stays readable after a provider drops that model (D-015).
- **Effort is baked into the model id**, so there is no effort control to build — eleven of agy's
  fourteen entries are the same three Gemini models at different efforts. The slider in his
  screenshot is agy's way of picking among them, not a second axis we have to model.
- **`agy` is a router, not a vendor.** It serves Gemini, Claude and GPT-OSS models. `Provider.routes`
  records it so nothing infers "who made this model" from which CLI answered.

**2 — done.** Marks for all three providers in `provider-icon.tsx`, drawn rather than imported:
they must tint from `currentColor` to work in provider, muted and disabled contexts, and DESIGN.md
forbids a hardcoded colour — which rules out agy's rainbow. They are recognisable approximations,
not official assets; swap all three at once if that is ever worth doing.

gemini is gone from `ProviderId`, the colour tokens, the mock, `Capabilities.md` and the blueprint,
and L-6 is deleted rather than left parked (D-014). **The ambiguous sentence, answered:** read as
*only include providers there is actually an account behind*. The cost is stated in D-014 — gemini
was the only live example of the three-state availability model, so `blocked` is now an unexercised
path. The state stays, because a signed-out or uninstalled CLI is the same shape.

**3 — already correct, nothing to change.** Verified in the browser rather than assumed: computed
`font-family` on `body` is `Geist, "Geist Fallback", …`, `Geist Mono` on code, and a Geist face
reports `loaded` in `document.fonts`.

**4 — done.** Provider dropdown then model dropdown, the second scoped to the first. Picking a
provider takes its default model. Changing model *within* a provider replays nothing — `seenBy` is
keyed by provider, and the provider's own session carries across models.

**5 — done.** Archive on the row menu, an Archived section in the sidebar, and **"Archive instead"**
in the delete dialog. Archiving is reversible so it gets a toast with Undo rather than a
confirmation; deleting is the one that asks. Search reaches into the archive — otherwise archiving
would make things unfindable, which is what deleting is for.

**6 — done.** Matches titles, folders and message text, and shows the matching line when the hit
was in the text. Verified against the case that justifies it: searching "quarter hour" finds an
**archived** conversation titled "Untitled conversation" by its message body — a title-only search
would have missed it completely.

### Two things this round turned up that were not asked about

- **`claude --bare` cannot authenticate this account.** It was recorded in the blueprint as
  MANDATORY scoping; its own `--help` says OAuth and the keychain are never read, and the owner
  signs in with a subscription. `claude --bare -p "hi"` → `Not logged in`. Shipping that adapter
  would have made claude unusable.

  **I reported this as unsolved, and the owner corrected it in one question:** *"then how does the
  multica able to use my claude code in this desktop... check what multica is doing."* Multica
  drives claude on this same machine, so a working answer already existed in a checkout we have
  read access to. It uses `--strict-mcp-config`. Verified here the same day —
  0 MCP servers, OAuth intact, and `--setting-sources project` cuts inherited skills 72 → 17. G-3 is
  **deleted, closed**; the recipe is in `Capabilities.md` and D-016. Two further findings came free
  from the same look: the daemon must scrub `CLAUDE_*`/`ANTHROPIC_*` or a nested `claude -p` hangs,
  and claude *can* enumerate its models via a `list_models` control request (unsupported on our
  2.1.90, working on 2.1.223+).

  **The lesson is about where I looked.** `Reference/multica-main` is a working implementation of
  this exact problem, and `AGENTS.md` §3 already says to consult it before inventing patterns. I
  read a `--help` string and concluded "unsolved" while a production answer sat unread in the repo.
  Read the reference before declaring a capability boundary.
- **B-1**, a real bug found in the browser: agy's 14-model menu rendered 175px below the fold
  because a `max-h-80` of mine overrode the popup's own `max-h-(--available-height)`. Fixed and
  re-verified at three viewport heights.

### What this told us about taste

One thing, and it is the same instinct as last round: **he checks the substance behind a surface,
not just its look.** Four of six items were about whether what is displayed is *true* — real model
names, real vendor marks, the real font — rather than about layout. The lesson for the next round
is that plausible filler is not a neutral choice: `agy-1` looked fine and was fiction, and it
survived a whole design round because nobody checked it against the CLI that was sitting right
there. Check the cheap facts before showing them.
