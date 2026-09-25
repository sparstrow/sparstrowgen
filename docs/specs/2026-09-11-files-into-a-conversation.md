# Spec: Put a file into a conversation

| | |
|---|---|
| **Status** | **Draft — needs your correction and approval** (revised 2026-09-24) |
| **Created** | 2026-09-11 |
| **Trigger** | "The task: file upload in the chat, drag and drop file" (2026-09-11); "I need to upload photos & file. I need a plus button in the prompt bar." (2026-09-24) |
| **Design** | not designed yet. **Your reference, 2026-09-24:** a plus button in the message box that opens a menu whose first entry is "Add photos & files" ([screenshot](../design/references/2026-09-24-prompt-bar-plus.png)). The rest of that reference (@ sources, / commands, dictation) is not part of this spec |
| **Open questions** | L-33, what the agents may do on your computer, decides which agents can open a file today |

> Drafted from one line, so the scenarios below are my guesses at what you meant. Correct them —
> especially US2 and US4, which I invented from how the app is used rather than from anything you
> said. **2026-09-24:** you have since said photos and files, from your computer, which answers most
> of question 2 below.

## What's wrong today

Everything an agent knows about has to already be in the project folder, or be typed into the
message. That is fine for code and impossible for everything else.

A screenshot of something rendering wrong cannot be described faster than it can be shown. A log
dump, a CSV, an error report, a spec document someone emailed — all of it currently has to be
pasted as text, which mangles anything that was not text to begin with and silently truncates
anything long. The alternative today is leaving the app: save the file into the project folder by
hand, then come back and type its path. That works, which is exactly why it is annoying — the app
is asking to be told something it could have seen.

## What I want instead

Drop a file onto the conversation and have the agent be able to use it, without me filing it
anywhere first.

## User stories

### US1 — Give an agent a file it could not otherwise see (P1)

**As** the owner **I want** to drop a file into a conversation and ask about it in the same
message, **so that** I can show something instead of describing it.

**Acceptance**

- **Given** a conversation is open, **when** I drop a file onto it and send a message, **then** the
  agent's reply shows it used the file's actual contents, not my description of it.
- **Given** I have dropped a file, **when** I look at the conversation afterwards, **then** it says
  which file I sent and with which message, and that is still true after a refresh.
- **Given** the file is an image, **when** the agent answers, **then** it can describe what is in
  the picture. *(Verified deliverable on `codex`; see Capabilities.)*
- **Given** I drop something the current agent cannot use, **when** I send it, **then** I am told
  that before I wait for an answer, not after.

### US2 — The file stays part of the conversation (P2)

*Guessed, not stated. Cut it if you only meant "send it once".*

**As** the owner **I want** a file I sent earlier to still be usable later in the same
conversation, **so that** referring back to it does not mean sending it again.

**Acceptance**

- **Given** I sent a file several messages ago, **when** I refer to it again, **then** the agent can
  still reach it.
- **Given** I switch the conversation to a different agent, **when** the new agent answers, **then**
  the file is as available to it as it was to the first — the file belongs to the conversation, not
  to whichever agent was answering when it arrived.

### US3 — Know what happened to it (P1)

**As** the owner **I want** to know where a file I dropped has gone, **so that** I am not
surprised later by files I did not knowingly put somewhere.

**Acceptance**

- **Given** I drop a file, **when** it has been sent, **then** I can tell that it now exists on my
  machine, and where — because a conversation is about a folder, and the agent can only read what
  is in it.
- **Given** I have sent several files over time, **when** I want them gone, **then** removing them
  is something I can do and understand, rather than a hidden store that only grows.

### US4 — It fails in the open (P1)

*The four states, which you have not seen for this feature yet.*

**As** the owner **I want** a failed send to be obvious and recoverable, **so that** I never
believe an agent has seen something it has not.

**Acceptance**

- **Given** my machine is unreachable, **when** I drop a file, **then** I am told the file cannot
  get there, and nothing pretends to have sent it.
- **Given** a file is too large, **when** I drop it, **then** I am told so at the moment I drop it,
  not after waiting for a transfer that was never going to finish.
- **Given** a send fails partway, **when** it stops, **then** the conversation does not contain a
  message claiming a file that is not there.

## Deliberately not in this

- **The agent sending a file back to me.** Multica has this — an agent that produces a chart or a
  report can attach it to its reply. Ours cannot, because that needs something on the machine the
  agent can call, which we do not have. Worth wanting; a separate piece of work.
- **A file library.** Files belong to the conversation that received them. Somewhere to browse
  everything ever sent is a different feature.
- ~~Anything on `agy` until B-11 is decided.~~ Struck 2026-09-24: agy now reads files (Capabilities,
  "Agent activity"). Which agents can open a file under today's settings is a feasibility question,
  answered in Capabilities before design, not here.

## What I need from you

1. **Is US2 right** — should a file stay usable for the rest of the conversation, or is one message
   enough?
2. **What do you actually drop?** Screenshots, logs, spreadsheets, documents, code from elsewhere?
   It changes what the transcript should show for one. *(2026-09-24: "photos & file". Still open:
   which kinds of file — PDFs, spreadsheets, logs?)*
3. **Anything here that is not what you meant**, including the whole framing.
