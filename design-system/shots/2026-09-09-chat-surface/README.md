# Chat surface — 2026-09-09

First shot round. Spec: [`docs/specs/2026-09-09-one-chat-across-providers.md`](../../../docs/specs/2026-09-09-one-chat-across-providers.md)
(Approved 2026-09-09).

Four directions, differing structurally rather than cosmetically — layout model, what dominates,
and interaction metaphor. Style, theme (dark), and viewport (desktop 16:10) held identical across
all four so the only variable is the idea.

| | Direction | Optimised for | Outcome |
|---|---|---|---|
| A | **Sidebar messenger** — conversation list left, thread right, per-message agent badges, headroom strip above the thread | Familiarity. Zero learning cost; looks like every chat app he already uses | **chosen** — "looks like claude code desktop which I prefer" |
| B | **Terminal log** — one full-width monospace transcript, no bubbles, no avatars; agents distinguished only by a coloured left rule, and the provider switch is a divider inside the record | Density and continuity. The switch becomes part of the transcript rather than a UI action | rejected as a shell, **switch divider adopted into A** — "that is brilliant too" |
| C | **Provider cockpit** — three provider cards with capacity gauges dominate the top, conversation demoted to a lower panel | Never being surprised by a limit. Treats "which agent, how much left" as the primary job | rejected — headroom belongs in a strip, not a dashboard |
| D | **Document with gutter** — one wide reading column, agent attribution in a margin gutter, floating toolbar | Reading a long working session back as a coherent artifact rather than a chat log | rejected as a shell, **centred agent column adopted into A** |

## What the feasibility gate required of all four

From [`docs/Capabilities.md`](../../../docs/Capabilities.md), every direction had to be able to
carry:

- **A "working" signal independent of text arriving** — `codex` provably never streams, so a
  direction whose only sign of life is text appearing would read as frozen on one of three
  providers.
- **Three-state provider availability with a reason** — available / waitable / blocked. `gemini` is
  installed-but-blocked today, so a dimmed-with-reason state is real, not hypothetical.
- **Asymmetric usage reporting** — `claude` reports real USD, `codex` and `agy` report tokens only.
  No direction may imply a single comparable number across all three.
- **Headroom only where it exists** — `rate_limit_event` is verified for `claude` alone.

## Notes on the generated images themselves

Recorded so the next round's prompts start better:

- **Greeked placeholder bars were wrong, and this round is why.** The first pass asked for neutral
  bars instead of text, to avoid garbled AI pseudo-words. It avoided them and produced four images
  the owner could not judge: *"I dont need empty image with boxes and charts, I want more real and
  acutal image."* Regenerated with real product names, real numbers, real message text and filler
  prose that reads as sentences. The skill's rule is reversed accordingly — imperfect text costs a
  glance, an abstract mockup costs the round.
- **Naming the product-specific elements works, and this round proved it.** The same generator, on
  the same machine, returned a generic Telegram clone when asked for "a chat app UI" during the
  capability test, and returned A — headroom strip, per-message agent badges, a three-dot provider
  selector, one bar greyed out for the blocked provider — when those elements were spelled out.
  Keep doing this; it is the difference between a useful round and a wasted one.
- **A took roughly three times as long as B/C/D** because it fell back to a slower generation path.
  Budget for one straggler in a parallel round rather than treating it as a failure.
- **Two prompt bugs, mine not the tool's.** (1) I wrote the word `dollar` instead of `$` to dodge
  shell escaping, and A, B and D rendered it literally as "dollar 0.03"; C alone interpreted it.
  Escape the `$` properly instead. (2) C's prompt enumerated only three provider cards — claude,
  codex, gemini — so `agy` is missing from that direction entirely. Take the roster from
  `blueprint.yaml`, not from memory.
- **Judge these on composition, hierarchy, density and whether the information shown is the right
  information.** They still do not show interaction, behaviour at real volume, or the four states —
  the prototype settles all of that.

## What this told us about taste

Durable preferences, in his words, not "he liked A":

- **The familiar shell wins over the clever one.** A was chosen because it "looks like claude code
  desktop which I prefer". Familiarity is a feature here, not a failure of imagination — he is
  replacing three apps he already knows, and a novel shell would be a fourth thing to learn.
- **A good idea can be extracted from a rejected direction.** He took B's switch divider and D's
  centred reading column into A's shell. Present directions as separable ideas rather than
  packages; expect the answer to be a merge.
- **Interface asymmetry carries meaning.** His messages right, agent output centred: "my replies
  should stay on right but the agent's reply should be centered." The agent's output is the thing
  being *read*, his own is just the prompt that caused it — so they should not look like two sides
  of a conversation of equals.
- **He thinks about what an interaction costs, not only what it does.** The single most substantive
  note in this round was not visual: replay must happen on the first prompt after switching, never
  at the moment of switching, "because switching models casually and using tokens or switching will
  burn lot of tokens." Expect him to catch cost-of-interaction problems; design for them up front.
- **He wants the real stack in front of him, not an approximation.** He stopped the build to ask
  whether shadcn was actually being used. A hand-rolled prototype that would later be rebuilt in
  the real components is waste he will notice and object to.

## What got built from this

Direction A, in the real app rather than as a standalone prototype: `apps/web`, Next.js + shadcn on
mock data. Merged in from the rejected directions: B's replay divider (made lazy) and D's centred
agent column. See [`docs/Decisions.md`](../../../docs/Decisions.md) D-013.
