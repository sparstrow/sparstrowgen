package protocol

// Files in a conversation (docs/specs/2026-09-11-files-into-a-conversation.md,
// D-056): what the owner uploads, what an agent makes, and the conversation's
// working folder read live through the daemon.

// MaxFileBytes is the largest file a conversation keeps, uploaded or made. The
// browser refuses a bigger one when it is added, the server refuses it when it
// arrives, and the daemon does not collect a bigger output.
const MaxFileBytes = 25 << 20

// MaxFolderFileBytes is the largest working-folder file the pane shows. It
// travels inside a websocket message, so it is kept well under what would stall
// a turn streaming on the same socket.
const MaxFolderFileBytes = 5 << 20

// FilesProtocol is the daemon protocol that understands files. A computer below
// it is told to update rather than sent a message with files it would ignore.
const FilesProtocol = 2

const (
	FileUpload = "upload"
	FileOutput = "output"
)

// ConversationFile is one file of a conversation.
type ConversationFile struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversationId"`
	// The user message it came with, or the agent turn that made it. Empty
	// while an upload waits in the message box.
	EntryID   string `json:"entryId,omitempty"`
	Origin    string `json:"origin"` // upload | output
	Name      string `json:"name"`
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	// RFC 3339, UTC.
	CreatedAt string `json:"createdAt"`
}

// TurnFile is a file the daemon must have on disk for a turn. Every file of the
// conversation is listed, so a conversation that moved to another computer gets
// its folder filled on the first turn there; the daemon downloads only what it
// is missing.
type TurnFile struct {
	ID     string `json:"id"`
	Origin string `json:"origin"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Sha256 string `json:"sha256"`
	// MediaType lets the daemon give pictures to codex through -i.
	MediaType string `json:"mediaType"`
	// Sent with the message this turn answers.
	Attached bool `json:"attached,omitempty"`
}

// FolderEntry is one thing inside a directory of the working folder.
type FolderEntry struct {
	Name string `json:"name"`
	// dir | file
	Kind string `json:"kind"`
	Size int64  `json:"size"`
	// RFC 3339, UTC.
	Modified string `json:"modified"`
}

// FolderListing answers ServerListFolder.
type FolderListing struct {
	// The directory listed, relative to the root, with forward slashes. Empty
	// is the root itself.
	Path    string        `json:"path"`
	Entries []FolderEntry `json:"entries"`
	// Why nothing could be listed, in words: the folder is gone, the path left
	// the root, it could not be read.
	Error string `json:"error,omitempty"`
}

// FolderFile answers ServerReadFile.
type FolderFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	// Base64 of the bytes. Empty when TooLarge or Error is set.
	Content string `json:"content,omitempty"`
	// Bigger than MaxFolderFileBytes, so not sent.
	TooLarge bool   `json:"tooLarge,omitempty"`
	Error    string `json:"error,omitempty"`
}

const (
	// ServerListFolder asks the daemon for one directory of a conversation's
	// working folder. Answered with DaemonFolderListing, echoing RequestID.
	ServerListFolder = "list_folder"
	// ServerReadFile asks the daemon for one file of a conversation's working
	// folder. Answered with DaemonFolderFile, echoing RequestID.
	ServerReadFile = "read_file"

	DaemonFolderListing = "folder_listing"
	DaemonFolderFile    = "folder_file"

	// EventFiles says a conversation's files changed: an upload, a removal, or
	// outputs a turn made. Files carries the entry's files when EntryID is set.
	EventFiles = "files"
)
