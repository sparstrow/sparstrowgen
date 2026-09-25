// Package protocol defines every shape that crosses a process boundary:
// server↔daemon and server↔browser.
//
// The JSON field names deliberately match apps/web/lib/chat-types.ts. Protobuf
// was deferred (docs/Decisions.md D-017), so until it lands these structs and
// that file are hand-kept in step, and this package is the side that owns the
// shape. Changing a json tag here without changing the TypeScript is the exact
// failure D-003 was worried about — keep them in the same commit.
package protocol

// ---------------------------------------------------------------------------
// Providers
// ---------------------------------------------------------------------------

// Availability is three-valued because "wait" and "act" are different
// instructions to the user. See docs/Decisions.md D-011.
type Availability string

const (
	Available AvailabilityValue = "available"
	// Waitable resolves itself: the machine is asleep, a limit window is
	// mid-reset. Nothing is broken.
	Waitable AvailabilityValue = "waitable"
	// Blocked will not resolve until a human does something — signs in,
	// installs the CLI.
	Blocked AvailabilityValue = "blocked"
)

// AvailabilityValue exists so the constants above type-check as Availability
// while remaining plain strings on the wire.
type AvailabilityValue = Availability

// Model always carries both halves. `agy models` prints
// `gemini-3.1-pro-high` next to "Gemini 3.1 Pro (High)"; the id is what the CLI
// is invoked with and the label is the only thing worth showing. Neither is
// derivable from the other — see docs/Decisions.md D-015.
type Model struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Headroom is what claude's rate_limit_event actually carries. Only claude
// emits one at all.
//
// There is NO percentage in it. The mock and the design shots both showed
// "62%" with a capacity bar, and that number was invented — the real payload is
// a status, a reset timestamp and a window name:
//
//	{"status":"allowed","resetsAt":1789083000,"rateLimitType":"five_hour", ...}
//
// So the surface can honestly say *when the window resets* and never *how much
// is left*. Time remaining is not capacity remaining, and a bar would be read
// as the second. See docs/Decisions.md D-018.
type Headroom struct {
	// "allowed" is the only value ever observed. See KnownGaps G-4.
	Status string `json:"status"`
	// Unix seconds. Sent raw so the client can count it down without the server
	// having to push a new value every minute.
	ResetsAt int64 `json:"resetsAt"`
	// The window this applies to, e.g. "five_hour".
	Window string `json:"window"`
}

type Provider struct {
	ID           string       `json:"id"`
	Label        string       `json:"label"`
	Models       []Model      `json:"models"`
	Model        *Model       `json:"model"`
	Availability Availability `json:"availability"`
	// Set whenever Availability is not "available".
	UnavailableReason string `json:"unavailableReason,omitempty"`
	// nil means the provider reports no limit signal at all. That is NOT the
	// same as having plenty left, and the UI must not render it as such.
	Headroom *Headroom `json:"headroom"`
	// Only claude reports real currency.
	ReportsUsd bool `json:"reportsUsd"`
	// Whether this provider emits a usage window at all. Distinct from Headroom
	// being nil: a provider that reports limits but has not been used yet this
	// session is "not known yet", which is not the same claim as "reports
	// nothing". Collapsing the two would tell the owner codex has no limits and
	// claude has none left, and both would be wrong.
	ReportsLimits bool `json:"reportsLimits"`
	// codex is verified false: one whole message per turn, no deltas.
	Streams bool `json:"streams"`
	// True when the CLI is a router in front of several vendors' models. agy
	// serves Gemini, Claude and GPT-OSS, so "provider" means the CLI we drive,
	// never who made the model.
	Routes bool `json:"routes"`
}

// ---------------------------------------------------------------------------
// Transcript
// ---------------------------------------------------------------------------

type Usage struct {
	Tokens int64 `json:"tokens"`
	// Omitted unless the provider states a real figure. Derived from
	// SpendTicks, which is the exact integer we store.
	Usd *float64 `json:"usd,omitempty"`
}

type Entry struct {
	ID   string `json:"id"`
	Role string `json:"role"` // user | agent | replay
	Seq  int32  `json:"seq"`
	// The instant this entry belongs to, as RFC 3339 in UTC — for an agent turn
	// that is when its answer arrived, not when the turn was launched. It is an
	// instant rather than a time of day because the server that formats it and
	// the person reading it are in different places: the server runs in UTC, so
	// a clock time formatted there showed the owner in Toronto 21:33 for a
	// message he sent at 17:33 (docs/Bugs.md B-37). The browser renders it in
	// whatever timezone the reader is actually in.
	At string `json:"at"`

	// user + agent
	Text string `json:"text,omitempty"`

	// agent + replay
	Provider string `json:"provider,omitempty"`
	Model    *Model `json:"model,omitempty"`

	// agent
	Usage   *Usage `json:"usage,omitempty"`
	Failure string `json:"failure,omitempty"`
	// Stopped is set when the owner ended the turn rather than the provider
	// finishing it. Separate from Failure because they are different events and
	// read differently: a failure is something going wrong, a stop is somebody
	// deciding they have seen enough. Both can be true at once when a CLI
	// complains on its way out.
	Stopped bool `json:"stopped,omitempty"`

	// user (sent with it) + agent (made in the turn)
	Files []ConversationFile `json:"files,omitempty"`

	// replay
	ToModel          *Model `json:"toModel,omitempty"`
	To               string `json:"to,omitempty"`
	MessagesReplayed int32  `json:"messagesReplayed,omitempty"`
	Tokens           int64  `json:"tokens,omitempty"`
}

// Workspace is a separate area of work inside an account (D-050). Role is this
// account's role in it, and it travels with the workspace because the surface
// that lists them is the same one that decides whether renaming is offered.
type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type Conversation struct {
	ID string `json:"id"`
	// Which workspace this conversation lives in. It travels with every
	// conversation because events are addressed to an ACCOUNT, not to a
	// workspace: without it, a tab looking at Personal would patch a Work
	// conversation into its own list the moment one changed.
	WorkspaceID string `json:"workspaceId"`
	// Empty when nobody has named it: not the owner, and not the first message
	// sent in it. The surface shows a placeholder, which describes a
	// conversation with no name rather than pretending to be one.
	Title    string  `json:"title"`
	Folder   string  `json:"folder"`
	Updated  string  `json:"updated"`
	Provider string  `json:"provider"`
	Model    Model   `json:"model"`
	SpendUsd float64 `json:"spendUsd"`
	Tokens   int64   `json:"tokens"`
	Archived bool    `json:"archived"`
	// Entries is nil in list responses and populated when one is opened.
	Entries []Entry `json:"entries"`
	// How much of the transcript each provider has already been told, so a
	// switch back replays only the gap.
	SeenBy map[string]int32 `json:"seenBy"`
	// Set only on a search result whose match was in the message text: the
	// matching line, so a hit in a long transcript is explicable. A title match
	// leaves this empty — the reason for that hit is already on screen.
	Excerpt string `json:"excerpt,omitempty"`
}

// ---------------------------------------------------------------------------
// Exchange: everything that crossed between the daemon and one CLI in one turn
// ---------------------------------------------------------------------------

// ExchangeSent is what the daemon handed the CLI, exactly. The environment is
// deliberately absent: it carries the sign-in token (CLAUDE_CODE_OAUTH_TOKEN),
// and a record meant for reading back must never hold a secret.
type ExchangeSent struct {
	// The resolved executable, so which install of a CLI ran is on record.
	Program string   `json:"program"`
	Args    []string `json:"args"`
	Cwd     string   `json:"cwd"`
	// Empty when the turn started a new provider session.
	ResumeSessionID string `json:"resumeSessionId,omitempty"`
	// The prompt as the agent reads it, catch-up included. Also inside Stdin,
	// which is how a CLI actually receives it; kept apart because the envelope
	// escapes every newline and a prompt is only readable without it.
	Prompt string `json:"prompt"`
	// The bytes written to the CLI's stdin. Empty when it never started.
	Stdin string `json:"stdin"`
	// False when the daemon refused the turn before starting the CLI: the
	// folder was missing, an update was installing, the stop overtook the start.
	// Prompt then says what would have been sent, and nothing else is claimed.
	Launched bool `json:"launched"`
}

const (
	StreamStdout = "stdout"
	StreamStderr = "stderr"
)

// ExchangeLine is one line a CLI printed, as printed.
type ExchangeLine struct {
	// Order of arrival at the daemon, from 1, per turn. The browser uses it to
	// notice a missed batch rather than trusting that none was.
	Seq int32 `json:"seq"`
	// Milliseconds since the CLI was started.
	AtMs   int64  `json:"atMs"`
	Stream string `json:"stream"`
	Text   string `json:"text"`
}

// ExchangeDropped counts what reached the daemon after a turn's record was full
// and was not kept. The record says so rather than ending quietly.
type ExchangeDropped struct {
	Lines int64 `json:"lines"`
	Bytes int64 `json:"bytes"`
}

// ExchangeUsage is a CLI's own account of the tokens one turn used, in one
// shape for all three. Input and Output are the whole of each; the optional
// fields are PARTS of them that a CLI broke out, and are absent when it did not.
type ExchangeUsage struct {
	Input     int64  `json:"input"`
	FromCache *int64 `json:"fromCache,omitempty"`
	ToCache   *int64 `json:"toCache,omitempty"`
	Output    int64  `json:"output"`
	Reasoning *int64 `json:"reasoning,omitempty"`
	Total     int64  `json:"total"`
}

// ExchangeReport is what a CLI said about itself during a turn. Derived from
// the stored lines each time it is read, never stored, so a better reading of
// an old turn's lines improves that turn too. A field a CLI does not report is
// empty, and the surface says so rather than guessing.
type ExchangeReport struct {
	CLIVersion     string         `json:"cliVersion,omitempty"`
	Model          string         `json:"model,omitempty"`
	PermissionMode string         `json:"permissionMode,omitempty"`
	Cwd            string         `json:"cwd,omitempty"`
	SessionID      string         `json:"sessionId,omitempty"`
	Tools          []string       `json:"tools,omitempty"`
	Skills         []string       `json:"skills,omitempty"`
	Agents         []string       `json:"agents,omitempty"`
	MCPServers     []string       `json:"mcpServers,omitempty"`
	Usage          *ExchangeUsage `json:"usage,omitempty"`
}

// ExchangeSummary is how big one turn's record is, without its contents, so a
// transcript can say which turns have one and how large before anything is
// opened.
type ExchangeSummary struct {
	EntryID  string           `json:"entryId"`
	Launched bool             `json:"launched"`
	Lines    int32            `json:"lines"`
	Bytes    int64            `json:"bytes"`
	LastAtMs int64            `json:"lastAtMs"`
	Dropped  *ExchangeDropped `json:"dropped,omitempty"`
}

// ContextDocument is one record an agent CLI keeps about its own context, as
// the daemon found it in that CLI's session store on the computer: a claude
// attachment, a codex rollout item, agy's generation input. Unchanged, so a
// better reading of it later improves old turns (docs/Capabilities.md, "What
// each agent loaded, word for word").
type ContextDocument struct {
	// Which CLI's record, and which kind of it: "claude.attachment",
	// "codex.session_meta", "codex.message", "codex.world_state",
	// "codex.turn_context", "agy.gen_metadata".
	Kind string `json:"kind"`
	// JSON for claude and codex; base64 of the protobuf for agy.
	Body string `json:"body"`
}

// ContextRecord is what the daemon read from a CLI's session store after a
// turn. Daemon to server only.
type ContextRecord struct {
	// The file it was read from, so the owner can open it himself.
	From string `json:"from,omitempty"`
	// Why nothing could be read: the file was not where this CLI keeps it, or
	// was not in the shape it had. The rest of the turn's record is unaffected.
	Error     string            `json:"error,omitempty"`
	Documents []ContextDocument `json:"documents,omitempty"`
}

// ContextPiece is one piece of what a CLI fed its model on a turn, read from
// the documents above: one section of its instructions, one tool, one skill,
// one instruction file.
type ContextPiece struct {
	// instructions | file | skill | tool | deferred | server | agent |
	// environment | other. A client that meets a kind it does not know shows it
	// under "other" rather than dropping it.
	Kind string `json:"kind"`
	Name string `json:"name"`
	// The file on the computer it came from, where the CLI recorded one.
	Source string `json:"source,omitempty"`
	Chars  int    `json:"chars"`
	// The CLI's own count of this piece's tokens. Only agy states one.
	Tokens *int64 `json:"tokens,omitempty"`
	Text   string `json:"text"`
}

// ExchangeContext is everything a CLI fed its model on one turn.
type ExchangeContext struct {
	From   string         `json:"from,omitempty"`
	Error  string         `json:"error,omitempty"`
	Pieces []ContextPiece `json:"pieces"`
	// What this CLI keeps no record of at all, in words: "codex does not record
	// its tool definitions." Said, so absence is never read as "none".
	Missing []string `json:"missing"`
}

// Exchange is one agent turn's record. As a live event it carries only what is
// new: the sent record once, then lines, then the report when it changes.
type Exchange struct {
	EntryID string `json:"entryId"`
	// False for a turn that has no record: it ran before recording existed, on
	// a daemon that does not record, or it never reached a computer.
	Recorded bool             `json:"recorded"`
	Sent     *ExchangeSent    `json:"sent,omitempty"`
	Lines    []ExchangeLine   `json:"lines"`
	Dropped  *ExchangeDropped `json:"dropped,omitempty"`
	Report   *ExchangeReport  `json:"report,omitempty"`
	// What the CLI fed its model, from its own session store. Absent for a turn
	// run by a daemon that did not read it.
	Context *ExchangeContext `json:"context,omitempty"`
}

// ---------------------------------------------------------------------------
// Server → daemon
// ---------------------------------------------------------------------------

const (
	// ServerRunTurn asks the daemon to run one turn against one CLI.
	ServerRunTurn = "run_turn"
	// ServerProbe asks the daemon to re-report what is installed.
	ServerProbe = "probe"
	// ServerStopTurn asks the daemon to end a turn that is already running,
	// killing the CLI and everything it spawned. Fire-and-forget, like
	// ServerRunTurn: the turn's actual end still arrives as DaemonStopped, so
	// there is one path for "this turn is over" rather than two.
	ServerStopTurn = "stop_turn"
	// ServerListDir asks the daemon what is inside a directory, and whether it
	// can be used as a conversation's working directory.
	//
	// This is the first request that expects an answer. Everything else here is
	// fire-and-forget, so it carries a RequestID the daemon echoes back — see
	// hub.Ask.
	ServerListDir = "list_dir"
)

type ServerMessage struct {
	Type string   `json:"type"`
	Turn *RunTurn `json:"turn,omitempty"`

	// stop_turn
	TurnID string `json:"turnId,omitempty"`

	// list_dir
	RequestID string `json:"requestId,omitempty"`
	// Empty asks for the starting points — drive roots on Windows, $HOME
	// elsewhere — because a browser has no idea what the machine looks like.
	// For list_folder and read_file, relative to Root.
	Path string `json:"path,omitempty"`

	// list_folder, read_file: the conversation's working folder. Nothing
	// outside it is listed or read.
	Root string `json:"root,omitempty"`

	// update_preference
	Automatic *bool `json:"automatic,omitempty"`
}

// ReplayEntry is one historical message being handed to a provider that has not
// seen it. Sent only when a catch-up is actually being paid for.
type ReplayEntry struct {
	Role     string `json:"role"`
	Provider string `json:"provider,omitempty"`
	Text     string `json:"text"`
	// Names of the files sent with a user message, or made in an agent turn,
	// so a provider catching up knows they exist and where.
	Files []ReplayFile `json:"files,omitempty"`
}

type ReplayFile struct {
	Origin string `json:"origin"`
	Name   string `json:"name"`
}

type RunTurn struct {
	TurnID         string `json:"turnId"`
	ConversationID string `json:"conversationId"`
	// EntryID is the agent entry already appended to the transcript. The daemon
	// streams into it, so a refresh mid-turn shows what has arrived so far.
	EntryID  string `json:"entryId"`
	Provider string `json:"provider"`
	Model    Model  `json:"model"`
	Cwd      string `json:"cwd"`
	Prompt   string `json:"prompt"`
	// Empty starts a fresh provider session. Set to continue one.
	ResumeSessionID string `json:"resumeSessionId,omitempty"`
	// Non-empty only when this turn is also catching a provider up.
	Replay []ReplayEntry `json:"replay,omitempty"`
	// Every file of the conversation, the ones sent with this message marked.
	// A daemon older than FilesProtocol ignores it, which is why the server
	// refuses to send files to one.
	Files []TurnFile `json:"files,omitempty"`
}

// ---------------------------------------------------------------------------
// Daemon → server
// ---------------------------------------------------------------------------

const (
	DaemonHello  = "hello"
	DaemonDelta  = "delta"
	DaemonDone   = "done"
	DaemonFailed = "failed"
	// DaemonLimit reports a provider's usage window. It arrives mid-turn,
	// because that is the only time a provider mentions one — there is no way
	// to ask.
	DaemonLimit = "limit"
	// DaemonStarted reports the provider's session id as soon as it is known,
	// so a resume pointer survives a turn that later fails.
	DaemonStarted = "started"
	// DaemonStopped reports a turn ended because it was asked to stop, as
	// distinct from failing. It carries whatever text had arrived, which is
	// usually the point: stopping is most often done once the answer has gone
	// somewhere useless, and what came before is still worth keeping.
	DaemonStopped = "stopped"
	// DaemonDirListing answers a ServerListDir, echoing its RequestID.
	DaemonDirListing = "dir_listing"
	// DaemonExchange carries part of a turn's record: first what was sent, then
	// the lines the CLI printed in batches, and anything dropped. Always before
	// the turn's done, failed or stopped. A server that predates it ignores it.
	DaemonExchange = "exchange"
)

// Why a directory is unusable, as a value rather than a sentence.
//
// The reason is rendered into a message by the surface that shows it, so the
// wording can change without the daemon knowing, and a client never has to
// parse prose to tell "you typed a file" from "that folder is read-only".
// Borrowed from Multica's local-directory validator.
const (
	DirOK           = ""
	DirNotAbsolute  = "not_absolute"
	DirNotFound     = "not_found"
	DirNotDirectory = "not_a_directory"
	DirNotReadable  = "not_readable"
)

// DirEntry is one subdirectory. Files are never listed: a conversation runs in
// a directory, so offering files would only invite an unusable choice.
type DirEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// DirListing is what the daemon knows about one directory.
type DirListing struct {
	// Path as the daemon resolved it — absolute, symlinks and "." collapsed —
	// so what gets stored is what the agent will actually run in.
	Path string `json:"path"`
	// Empty at a root. Lets the picker walk up without doing path arithmetic in
	// the browser, where the separator may not even match the machine.
	Parent string `json:"parent"`
	// Never null: emit_empty_slices is the Go side, and the picker renders a
	// real "nothing in here" rather than a missing state.
	Entries []DirEntry `json:"entries"`
	// DirOK, or one of the reasons above.
	Reason string `json:"reason"`
	// Whether the directory sits in a git working tree. Advisory only — it is
	// shown so a folder that is not the project root is noticeable before a
	// conversation starts, never to prevent the choice.
	IsGitRepo bool `json:"isGitRepo"`
}

type DaemonMessage struct {
	Type string `json:"type"`

	// hello
	Machine   string     `json:"machine,omitempty"`
	Providers []Provider `json:"providers,omitempty"`
	// Absent from daemons older than US3, which is read as protocol 0, no
	// version, and no updater — never guessed.
	Version     string `json:"version,omitempty"`
	Protocol    int    `json:"protocol,omitempty"`
	SelfUpdates bool   `json:"selfUpdates,omitempty"`
	// Where this computer keeps each conversation's files: <ChatsDir>/<id>/
	// uploads and outputs. Absent from a daemon older than FilesProtocol.
	ChatsDir string `json:"chatsDir,omitempty"`

	// update_status
	Update *UpdateStatus `json:"update,omitempty"`

	// everything else
	TurnID string `json:"turnId,omitempty"`

	// delta
	Text string `json:"text,omitempty"`

	// started
	SessionID string `json:"sessionId,omitempty"`

	// limit
	Provider string    `json:"provider,omitempty"`
	Headroom *Headroom `json:"headroom,omitempty"`

	// done
	Full       string `json:"full,omitempty"`
	Tokens     int64  `json:"tokens,omitempty"`
	SpendTicks int64  `json:"spendTicks,omitempty"`

	// failed
	Error string `json:"error,omitempty"`

	// dir_listing
	RequestID string      `json:"requestId,omitempty"`
	Listing   *DirListing `json:"listing,omitempty"`

	// folder_listing, folder_file
	Folder *FolderListing `json:"folder,omitempty"`
	File   *FolderFile    `json:"file,omitempty"`

	// exchange
	Sent    *ExchangeSent    `json:"sent,omitempty"`
	Lines   []ExchangeLine   `json:"lines,omitempty"`
	Dropped *ExchangeDropped `json:"dropped,omitempty"`
	// Once, after the last lines and before the turn's ending.
	Context *ContextRecord `json:"context,omitempty"`
}

// ---------------------------------------------------------------------------
// Server → browser
// ---------------------------------------------------------------------------

const (
	EventProviders    = "providers"
	EventEntryAdded   = "entry_added"
	EventEntryDelta   = "entry_delta"
	EventEntryDone    = "entry_done"
	EventConversation = "conversation"
	// EventDaemon reports whether the machine is reachable at all. Everything
	// already said stays readable when it is not; only sending is impossible.
	EventDaemon   = "daemon"
	EventMachines = "machines"
	// EventAppearance says this account's saved appearance changed, so every
	// other browser it is signed in to converges on it rather than keeping the
	// look it happened to load with.
	EventAppearance = "appearance"
	// EventProfile says this account's name, description or picture changed.
	// Same reasoning as EventAppearance: the name is shown in the shell on every
	// page, so a tab that was open while it changed would keep showing the old
	// one until it happened to be reloaded.
	EventProfile = "profile"
	// EventWorkspaces says the list of workspaces this account can reach has
	// changed — one was created or renamed. The list itself is not sent: the
	// browser refetches it, because unlike a conversation there is no partial
	// update worth patching in and the list is three rows long.
	EventWorkspaces = "workspaces"
	// EventExchange carries what is new in a running turn's record, so an open
	// Raw view grows while the agent works.
	EventExchange = "exchange"
)

type ClientEvent struct {
	Type           string        `json:"type"`
	ConversationID string        `json:"conversationId,omitempty"`
	Providers      []Provider    `json:"providers,omitempty"`
	Entry          *Entry        `json:"entry,omitempty"`
	EntryID        string        `json:"entryId,omitempty"`
	Text           string        `json:"text,omitempty"`
	Conversation   *Conversation `json:"conversation,omitempty"`
	Online         bool          `json:"online,omitempty"`
	// With EventDaemon: the connected computer is too old to be sent work, so
	// the surface says to update it rather than offering a send that will fail.
	TooOld bool `json:"tooOld,omitempty"`
	// With EventExchange.
	Exchange *Exchange `json:"exchange,omitempty"`
	// With EventFiles and an EntryID: that entry's files, all of them.
	Files []ConversationFile `json:"files,omitempty"`
}

// SpendTicksPerUSD is the fixed-point scale for money. Cost is stored and moved
// as an integer number of ticks, never a float: a provider's own cost figure is
// authoritative and re-deriving it from tokens times a rate cannot reproduce
// request-level pricing rules. Adopted from Multica's TokenUsage.CostUSDTicks.
const SpendTicksPerUSD = 10_000_000_000

func TicksToUSD(ticks int64) float64 {
	return float64(ticks) / float64(SpendTicksPerUSD)
}

func USDToTicks(usd float64) int64 {
	return int64(usd * float64(SpendTicksPerUSD))
}
