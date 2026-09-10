# Spec: One chat across every installed agent

| | |
|---|---|
| **Status** | Draft |
| **Created** | 2026-09-09 |
| **Trigger** | "when I am building an app, I am juping between different desktop app chat window. I want one chat window where I can have the chat conversation to be stored, when I runout of limit on one I want to switch the conversation and continue with another provider and model" |
| **Design** | not designed yet |
| **Open questions** | G-4 (whether a provider can tell us it is out of limit) |

## What's wrong today

Three separate chat applications are open while working on one app — Claude Code, Codex, and Agy.
The work is one continuous train of thought, but it is split across three places that know nothing
about each other.

When one runs out of limit, the conversation stops dead. Carrying on means opening a different app
and re-explaining everything that was already said: what the app is, what was tried, what went
wrong, what was decided. The explaining costs more than the original question did, and the thread
of thought is lost in the middle of doing something.

The three tools also can't be compared. There is no way to see that one has plenty of limit left
while the one being used is nearly out — that only becomes apparent by being cut off.

## What I want instead

One place I talk to whichever agent I have installed, and the conversation belongs to *me*, not to
whichever tool answered. When one runs out, I pick a different one and keep going in the same
conversation — it already knows everything that was said, because the conversation was never that
tool's to begin with. And I can tell before I'm interrupted that I'm getting close.

## User stories

### US1 — Have a conversation with an installed agent (P1)

**As** the owner **I want** to send a message to one of the agents installed on my machine and get
its reply, in a conversation tied to the project I'm working on, **so that** I can work with an
agent without opening its own application.

**Acceptance**

- **Given** at least one agent is usable on my machine, **when** I start a new conversation and
  choose which project folder it is about, **then** I can send a message and see the reply arrive.
- **Given** an agent is replying, **when** it is taking a while, **then** I can tell it is still
  working and has not silently failed — and this is true whether or not its reply is arriving
  gradually.
- **Given** a reply has finished, **when** I look at it, **then** I can tell which agent answered
  and what it cost me — in money if that agent reports money, otherwise in whatever it does report.
- **Given** I have had a conversation, **when** I close the app entirely and come back later,
  **then** the whole conversation is still there exactly as it was.
- **Given** my machine is asleep or the connection to it is down, **when** I try to send a message,
  **then** I am told plainly that my machine is unreachable, and my message is not silently lost.
- **Given** an agent fails partway through a reply, **when** it stops, **then** I can see what was
  said before it broke and what went wrong, and the conversation is still usable afterwards.

### US2 — Continue the same conversation on a different agent (P1)

**As** the owner **I want** to switch a conversation to a different agent and model and carry on,
**so that** running out of limit on one costs me a click rather than re-explaining everything.

**Acceptance**

- **Given** a conversation with history on one agent, **when** I switch it to a different agent and
  send my next message, **then** that agent answers with full knowledge of everything already said,
  without me pasting or re-explaining anything.
- **Given** I have switched agents, **when** I read back through the conversation, **then** it reads
  as one continuous conversation, and I can still tell which agent said which part.
- **Given** a long conversation, **when** I switch to an agent that has not seen it, **then** I know
  in advance that catching it up is not free, and roughly what it will cost, before I commit.
- **Given** I switch back to an agent that saw the earlier part of the conversation, **when** I send
  a message, **then** it only needs catching up on what it missed, not the whole thing again.
- **Given** an agent is installed but I have no account signed into it, **when** I look at what I can
  switch to, **then** it appears with the reason it is unavailable rather than being hidden — so I
  know it exists and what would make it work.
- **Given** the agent I want to switch to is not installed at all, **when** I look at my choices,
  **then** it simply isn't offered.

### US3 — See where each agent stands before I'm interrupted (P2)

**As** the owner **I want** to see how much room each agent has left, **so that** I switch on my own
terms instead of finding out by being cut off mid-thought.

**Acceptance**

- **Given** I am working in a conversation, **when** the agent I'm using is getting close to its
  limit, **then** I can see that, and when its limit next resets.
- **Given** an agent cannot tell us anything about its limits, **when** I look at it, **then** it is
  honest about that rather than showing a reassuring guess.
- **Given** I have been working for a while, **when** I look at a conversation, **then** I can see
  what it has cost me so far across every agent that worked on it.

### US4 — Come back to earlier conversations (P2)

**As** the owner **I want** to find and reopen a conversation I had before, **so that** work on a
project survives closing the app and I am not starting over each session.

**Acceptance**

- **Given** I have had several conversations, **when** I come back, **then** I can find the one I
  want and pick up where I left off.
- **Given** I have never used the app before, **when** I open it, **then** it is obvious what to do
  first rather than looking broken or empty.

## Edge cases

- **No agent is usable at all** — none installed, or none signed in. The app must say so and say
  what would fix it. It must not present a chat box that cannot work.
- **My machine is off.** Everything about a conversation that already happened is still readable;
  only sending something new is impossible, and that must be obvious before I type, not after.
- **An agent goes quiet for a long time.** Some agents send their reply gradually and some send
  nothing at all until the whole answer is ready. Both must look like *working*, never like frozen.
- **Switching in the middle of a reply.** Either it finishes first or it is clearly abandoned. It
  must not end up half-answered by one agent and half by another with no indication.
- **A conversation gets very long.** Catching a new agent up gets more expensive the longer it gets.
  I should not discover this as a surprise charge.
- **The same conversation open twice** — two browser tabs, or a phone and a laptop. It must not end
  up with two half-conversations that disagree.
- **An agent replies with something enormous.** A very long answer must not make the conversation
  unusable to scroll or read.

## Out of scope

- **Switching agents automatically when one runs out.** Wanted, but we cannot yet detect "out of
  limit" reliably on every agent (`KnownGaps.md` G-4). Switching stays my decision for now — and
  US3 exists so I can make it in time.
- **Gemini.** Installed but I have no account signed in (`Later.md` L-6).
- **Approving individual actions before an agent takes them.** Agents work auto-approved inside
  folders I have registered (`Decisions.md` D-007); the approval flow is a later addition
  (`Later.md` L-4).
- **Seeing the app the agent just built, or my desktop.** Separate features — the preview tunnel,
  and `Later.md` L-1.
- **Searching across all my conversations by meaning.** Comes with embeddings and vector search
  later; this spec only needs to find a conversation again, not search inside all of them.
- **A phone app or a desktop app.** The web app is the only surface here (`Later.md` L-2, L-3).
