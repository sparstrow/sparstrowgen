# Files in a conversation — handoff

| | |
|---|---|
| **Prototype** | `files.dc.html` (pictures in `files-assets/`) |
| **Provenance** | [`docs/specs/2026-09-11-files-into-a-conversation.md`](../../../specs/2026-09-11-files-into-a-conversation.md), approved 2026-09-24 |
| **Mode** | build: the plus button, the right-hand pane behind a folder icon, its uploads/outputs and working-folder browser, and previews were the owner's own calls |
| **Status** | built into the app without a separate review, while the owner was away (the approval asked for the feature to be carried through); his review comes on the real thing |
| **Design system** | the Claude artifact linked from `CLAUDE.md` |

## What this is

Pictures and files go into a message with the plus button, by dropping them on the conversation, or
by pasting. They are kept in the chat's own folder on the computer, so any agent in the conversation
can use them later. Pictures an agent makes appear under its answer. A pane on the right, opened by a
folder icon in the header, holds two things: **This chat** (uploads and outputs) and **Working
folder** (the conversation's folder on the computer, browsed live). Either one opens a preview.

## Component mapping

| Prototype element | Use | Notes |
|---|---|---|
| Plus button, its menu | existing `Button` (ghost, icon) + `DropdownMenu` | One entry today, "Add photos & files"; the menu is where @ sources and / commands go later |
| Attachment chips in the box | **new** `AttachmentChip` | picture thumbnail or kind icon, name, size or progress, remove |
| Drop overlay | **new**, inside the chat's main column | only while a drag carrying files is over it |
| Files in a message / under an answer | **new** `FileTile` (picture) and `FileCard` (other) | both open the preview in the pane |
| Folder icon in the header | existing icon `Button`, pressed state | shows the count of the chat's files while the pane is closed |
| Pane | **new** `FilesPane` | 400px beside the chat at desktop widths; over the chat below 1024px |
| Tabs in the pane | existing `ChoiceToggle` | "This chat" / "Working folder" |
| Rows, breadcrumbs, preview header | **new**, in `components/chat/files-pane.tsx` | |
| Markdown preview | existing `Markdown` | |

## Token usage

Only existing tokens. The pane uses `--background`, its header `--border`; previews of pictures sit
on `--muted`. Nothing new.

## States

| State | Reachable | Notes |
|---|---|---|
| This chat: populated / empty / loading / error | `?files=` | The empty text says how files get here |
| Working folder: populated / empty / loading / computer offline / too old | `?folder=` | Offline points back to the chat's own files, which do not need the computer |
| Box: nothing / attached / uploading / failed / too large | `?comp=` | Send waits for uploads; a failed one must be retried or removed |
| What the agent can open | `?provider=codex` | Said before sending, not after |
| Preview: picture, CSV, code/text, markdown, PDF, no preview | open any file | PDF uses the browser's own viewer; "no preview" offers the download |

## Data contract

**Files on the server.** A file is a row of the conversation, with its bytes stored beside it:

| Field | Type | Source |
|---|---|---|
| `id` | uuid | server |
| `conversationId` | uuid | |
| `entryId` | uuid, null while it waits in the box | the message it was sent with, or the agent turn that made it |
| `origin` | `upload` \| `output` | |
| `name` | string | the file's own name; a clash in the folder gets ` (2)` |
| `mediaType` | string | sniffed by the server, never trusted from the browser |
| `size` | int | bytes, 25 MB at most |
| `createdAt` | RFC 3339 | |
| `path` | string, empty until the computer has it | the file's full path on the computer |

- `GET /api/conversations/{id}/files` lists them, newest first. `POST` the same path uploads one (the
  body is the bytes); it answers with the row, `entryId` null. `GET /api/files/{id}/content` returns the
  bytes; `DELETE /api/files/{id}` removes a file not yet sent.
- A message carries `fileIds`; sending attaches them to its entry. An entry carries `files` (the rows
  above) so the transcript can show them. Outputs arrive with an `entry_files` event when the turn ends.

**On the computer**, per conversation: `<sparstrowgen data folder>\chats\<conversation id>\uploads`
and `\outputs`. Before a turn the daemon downloads what the message carries (HTTP, with its machine
credential), writes it into `uploads`, and names the paths in the prompt. After the turn it collects
new files from `outputs` and from the agent's own picture folder, copies those into `outputs`, and
uploads them.

**The working folder** is read live through the daemon: a listing of one directory (name, kind,
size, modified) and one file's bytes, 5 MB at most, both confined to the conversation's folder.
Nothing about it is stored.

Every row is deliverable: see Capabilities, "Files the owner drops into a conversation", re-checked
2026-09-24.

## Interactions

- Plus opens the menu; "Add photos & files" opens the system file dialog (several at once).
- Dropping files on the conversation, or pasting a picture into the box, adds them the same way.
- A file starts uploading as soon as it is added; Send waits until every file is up.
- A file over 25 MB is refused when it is added, with its name and size.
- Clicking a file anywhere (message, answer, pane) opens it in the pane's preview.

## Invented

- The 25 MB limit per file.
- The pane's width (400px) and "Wider" for a preview.
- The tab names "This chat" and "Working folder".
- The count badge on the folder icon.
- Text files for codex: small ones (100 KB or less) are pasted into its prompt; anything else it
  cannot open is named in a warning before sending.
- Files are kept in the database as well as on the computer, so the chat still shows them when the
  computer is off.

## Not included

- @ sources, / commands, dictation: later, as the owner said.
- Deleting a file after it was sent. The spec wants removal to be understandable; the pane shows the
  folder's path so it can be found, and removal is a follow-up (G-50).
- Agents' non-picture outputs made outside `outputs`: a file an agent writes into the project stays
  there, as the owner asked, and is seen under Working folder.

## Verification

2026-09-24, served locally in the Browser pane at 1440×900: every toggle in the bar, the plus menu
opening and closing, a CSV and an `.xlsx` opened from the pane, a code file opened from Working folder
after navigating two folders down, back to the same folder. No console errors.
