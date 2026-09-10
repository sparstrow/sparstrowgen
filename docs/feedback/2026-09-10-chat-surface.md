# Chat surface — feedback round, 2026-09-10

First review of the chat surface built in `apps/web` (commit `36ece44`, direction A from the
[2026-09-09 shot round](../../design-system/shots/2026-09-09-chat-surface/README.md)).

Captured live during the round, in his words. **Status: open — round still in progress.**
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

*Pending — the owner has not finished the round.*

## Outcomes

*Pending.*
