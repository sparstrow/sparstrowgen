# Spec: Put a file into a conversation

| | |
|---|---|
| **Status** | **Approved 2026-09-24** |
| **Created** | 2026-09-11 |
| **Trigger** | "The task: file upload in the chat, drag and drop file" (2026-09-11); "I need to upload photos & file. I need a plus button in the prompt bar." (2026-09-24) |
| **Design** | `docs/design/prototypes/Chat/files.dc.html`. **Your own design calls, 2026-09-24:** a plus button in the message box; a right-hand side pane opened by a new folder icon, showing the conversation's uploads and outputs, a preview of the selected file, and a browser of the working folder. **Your reference:** a plus button in the message box that opens a menu whose first entry is "Add photos & files" ([screenshot](../design/references/2026-09-24-prompt-bar-plus.png)). The rest of that reference (@ sources, / commands, dictation) is not part of this spec |
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

## Your answers (2026-09-24)

> "Spec approved.
> 1. Yes, whole conversation. The file that I upload needs to be in upload folder, and the incase if
> the agent generate a file like photo, where codex and agy can generate. I would like to see them in
> the chat and also needs to see the side pane. New folder icon for the right side pane where uploads
> and outputs will be stored. The outputs. Not all the outputs are stored in that directory. For
> example when I ask to edit or create a file, or give a coding task, the files needs to be saved in
> the appropriate location as usual by agent. I expect that to be an behaviour.
> 2. Mostly all file types, I expect a right side pane to also able to view their preview of the file.
> Also file browser of the current working directory should also be there.
> 3. Those we can add later."

So US2 stands, and three stories follow from the answers.

### US5 — See what an agent made (P1)

**As** the owner **I want** a file an agent generates for me, such as a picture, to appear in the
conversation and alongside my uploads, **so that** I do not have to go looking for it on disk.

**Acceptance**

- **Given** I ask codex or agy for a picture, **when** the turn ends, **then** the picture is in the
  conversation, with the answer that produced it, and it is among the conversation's files.
- **Given** I ask an agent to create or change a file that belongs to my project, **when** it does,
  **then** that file is where the agent put it in my project, as it would be without sparstrowgen,
  and it is not copied into the conversation's files.
- **Given** the agent made nothing, **when** the turn ends, **then** nothing is added.

### US6 — Look at a file without leaving the conversation (P1)

**As** the owner **I want** to see what is in any file the conversation holds, **so that** I can check
it while I talk about it.

**Acceptance**

- **Given** a picture, a PDF, a text or code file, a spreadsheet export or a log, **when** I open it,
  **then** I see its contents.
- **Given** a file that cannot be shown, **when** I open it, **then** I am told so and can still
  download it.

### US7 — See the folder the conversation works in (P2)

**As** the owner **I want** to browse the conversation's working folder and look inside its files,
**so that** I can see what the agent is working on, including what it just changed.

**Acceptance**

- **Given** a conversation with a working folder, **when** I browse it, **then** I see its folders and
  files as they are on my computer now, and can open one to see its contents.
- **Given** my computer is unreachable, **when** I browse, **then** I am told so, and the
  conversation's own files are still there to look at.
- **Given** a very large file, **when** I open it, **then** I am told it is too large to show here.

## What I needed from you (answered above)

1. **Is US2 right** — should a file stay usable for the rest of the conversation, or is one message
   enough?
2. **What do you actually drop?** Screenshots, logs, spreadsheets, documents, code from elsewhere?
   It changes what the transcript should show for one. *(2026-09-24: "photos & file". Still open:
   which kinds of file — PDFs, spreadsheets, logs?)*
3. **Anything here that is not what you meant**, including the whole framing.
