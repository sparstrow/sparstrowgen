# Spec: See the raw exchange with the agent

| | |
|---|---|
| **Status** | **Draft** — for the owner to correct and approve |
| **Created** | 2026-09-23 |
| **Trigger** | "I want to see what actually being sent raw to agent and what the agent replied exactly and everything." Why: "Rendered is formatted chat, but I also want to see if we rendered properly from AI, and also what see what are the context AI agent are feeding, reading, thinking etc." |
| **Design** | not designed yet |
| **Related** | [See what the agent did](2026-09-24-agent-activity.md) — the everyday, tidied view of the same turn. This one is the unedited record. Both need the same thing kept from every turn. |
| **Open questions** | none parked; see the edge cases marked *(inferred)* |

> Drafted from your message. Lines marked *(inferred)* are my guesses. Correct anything that's wrong.

## What's wrong today

The Raw view prints the text we saved for each message, character for character. That answers one
question: did the formatting drop or mangle something the agent wrote? It answers nothing else.

**It does not show what the agent was given.** After a switch, the new agent receives the earlier
conversation wrapped in instructions we wrote. Raw shows only "replayed 2 messages · 361 tokens".
No view shows the model, the folder, whether the agent continued its own earlier session or started
over, or what it was allowed to do.

**It does not show what the agent sent back.** An agent reports far more than its answer: that it
thought, each file it read and what was in it, each command and its output, each thing it was
refused, its own warnings, and how much it read. We keep the answer and the token count and throw
the rest away. Nothing thrown away can be shown later, in any view.

**It does not show the context the agent loaded by itself.** A one-line message can cost 40,000
tokens, because the agent reads its own instructions, tools and project files before it answers.
Today the only trace of that is the token count.

## What I want instead

For any message I sent, I can see exactly what went to the agent and exactly what came back,
unedited and in order. That lets me tell a rendering problem from an agent problem, see what the
agent was working from, and see what it actually did. Where an agent doesn't report something, I'm
told it wasn't reported. I'm never shown a guess.

## User stories

### US1 — What was sent (P1)

**As** the owner **I want** to see exactly what the agent received for a message **so that** I know
what it was working from.

**Acceptance**

- **Given** a message in a conversation that never switched agent, **when** I look at what was
  sent, **then** I see my words as the agent received them, plus the agent and model, the folder,
  whether it continued its earlier session or started fresh, and what it was permitted to do.
- **Given** the first message after a switch, **when** I look at what was sent, **then** I see the
  whole catch-up: the instructions we added, then every earlier message as the new agent received
  it, then my message.
- **Given** a message that never reached the agent (computer offline, folder missing, stopped
  before it started), **when** I look, **then** I see what would have been sent and why it wasn't.

### US2 — What came back, unedited (P1)

**As** the owner **I want** everything the agent reported during the turn, in the order it arrived
and unchanged, **so that** I can check it against the chat and check both the rendering and the
agent.

**Acceptance**

- **Given** a finished turn, **when** I look at what came back, **then** everything the agent
  reported is there, in order: its messages, each time it thought, each tool it used with what it
  asked for and what it got, each refusal with its reason, its own warnings and errors, and its
  usage.
- **Given** a turn that failed or that I stopped, **then** everything up to that moment is there,
  including the agent's last error in its own words.
- **Given** a turn still running, **when** I look, **then** what has arrived so far is there, and
  more appears as the agent works. *(inferred: this is what would show whether a slow turn is
  stuck)*
- **Given** a turn from before this existed, **then** I'm told the raw exchange was not recorded
  for that turn. The saved answer is never passed off as the raw exchange.

### US3 — What the agent loaded by itself (P2)

**As** the owner **I want** to see what the agent brought into the turn itself **so that** I can
explain a large token count, or an answer based on something I never said.

**Acceptance**

- **Given** a finished turn, **then** I can see what the agent reported loading: its version and
  model, its tools, skills and settings, and the folder it worked in. I can also see how much it
  read in total next to how much I sent.
- **Given** an agent that reports little or nothing about what it loaded, **then** that is said
  plainly for that agent, not left blank.

## Edge cases

- **Thinking.** When an agent reports that it thought but not what it thought, the record shows
  only that. The thoughts are never reconstructed. Today none of the three sends the words.
- **Very large turns.** An agent that reads a 5,000-line file sends the whole file back. It is kept
  and shown in full. If anything ever has to be cut, the record says where and how much was
  cut. *(inferred)*
- **How long it is kept.** For as long as the conversation exists, and it is deleted with the
  conversation. The point is checking something later, like an answer you only notice is wrong next
  week. *(inferred)*
- **What the agent read is kept too.** That includes a file of passwords in the project folder, if
  the agent opened it. It lives on the server with the rest of the conversation, which already
  holds anything the agent quoted in its answers. *(inferred: say so if that is not acceptable)*
- **Another device, another day.** You see the same record.
- **Both views describe one turn.** The tidied step view and the raw record never disagree about
  what happened.

## Out of scope

- **The agents' built-in instructions, word for word.** No agent reports them; only that they were
  loaded, and how large they were.
- **Thinking text**, for the same reason.
- **Editing what was sent and sending it again.** A possible later idea, not part of this.
- **The everyday step-by-step view.** That is [See what the agent did](2026-09-24-agent-activity.md).
