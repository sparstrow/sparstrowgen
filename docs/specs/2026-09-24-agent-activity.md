# Spec: See what the agent did

| | |
|---|---|
| **Status** | **Draft** — for the owner to correct and approve |
| **Created** | 2026-09-24 |
| **Trigger** | "When prompt is sent in the chat, I want all different process agent go through should be visible to me, like the thinking, searching, file edited, terminal commands ran etc." and "After the work is done by an agent, I want to see all these agents works to be collapsed … then I should only see the final response from the agent. If I want to I'll open what it worked." Then, 2026-09-24: "I would like to see the code diff … when a turn of work is done editing any file. And also I need side pane for a full view of changes that has made on the current session." |
| **Design** | [`docs/design/prototypes/Chat/agent-activity.dc.html`](../design/prototypes/Chat/agent-activity.dc.html), from the owner's reference component and screenshot |
| **Open questions** | [`docs/Later.md`](../Later.md) L-33: what the agents may do on the computer |

> Drafted from your two messages. It says what you need to be able to do and what is true
> afterwards. What it looks like is in the prototype, which is where you correct it.

## What's wrong today

While an agent works, the chat shows that it is working and for how long, and nothing else. When
it finishes, only its answer is shown. What it read, changed, ran or looked up is invisible. An
answer that says "I fixed it" cannot be checked against what was actually done, and a turn that
takes two minutes gives no sign whether it is making progress or stuck.

## Scenarios

**1. Watching a turn.** You send a message. While the agent works, each thing it does appears as
it happens: thinking, reading a file, searching the code or the web, editing a file, running a
command. You can tell which step it is on now.

**2. Reading the answer.** When the turn ends, the steps fold away. You see one line saying the
agent worked, and for how long, followed by its final answer. The answer is what you read by
default.

**3. Checking the work.** You open the folded line and see every step in order. For a file edit
you can see what changed. For a command you can see what it printed. You close it again.

**4. When the agent was not allowed.** If the agent tried something it was not permitted to do,
that shows as a step, with the reason, not hidden. You can tell "it did not try" from "it tried
and was refused".

**5. When the turn broke.** If the turn fails or you stop it, the steps it had already taken are
kept and can be opened.

**6. Coming back later.** A conversation opened tomorrow, or on another device, shows the same
folded line and the same steps for every turn.

**7. Seeing what a turn changed.** When a turn has created, edited or deleted files, the finished
turn shows the changes themselves, line by line, without opening anything: which lines were removed,
which were added, and where in the file.

**8. Reviewing everything a conversation changed.** Next to the conversation, you can look at every
change its turns made, file by file, and narrow it to one turn. You can keep it open while you read
and while the next turn runs.

## What is true afterwards

- Nothing an agent reports about its work is hidden from you, and nothing it did not report is
  shown as if it had.
- Every agent's steps look the same, although each agent reports them differently. A difference
  in what an agent *can* report (for example, codex reports a web search without its sources) is
  shown as absent, not invented.
- A turn that only answered shows no activity line at all.
- A change counts whoever made it, the agent's edit or a command it ran. A change you made yourself
  between turns is not shown as the agent's.
