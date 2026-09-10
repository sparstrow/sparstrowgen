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
	At   string `json:"at"`

	// user + agent
	Text string `json:"text,omitempty"`

	// agent + replay
	Provider string `json:"provider,omitempty"`
	Model    *Model `json:"model,omitempty"`

	// agent
	Usage   *Usage `json:"usage,omitempty"`
	Failure string `json:"failure,omitempty"`

	// replay
	ToModel          *Model `json:"toModel,omitempty"`
	To               string `json:"to,omitempty"`
	MessagesReplayed int32  `json:"messagesReplayed,omitempty"`
	Tokens           int64  `json:"tokens,omitempty"`
}

type Conversation struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Folder   string `json:"folder"`
	Updated  string `json:"updated"`
	Provider string `json:"provider"`
	Model    Model  `json:"model"`
	SpendUsd float64 `json:"spendUsd"`
	Tokens   int64  `json:"tokens"`
	Archived bool   `json:"archived"`
	// Entries is nil in list responses and populated when one is opened.
	Entries []Entry `json:"entries"`
	// How much of the transcript each provider has already been told, so a
	// switch back replays only the gap.
	SeenBy map[string]int32 `json:"seenBy"`
}

// ---------------------------------------------------------------------------
// Server → daemon
// ---------------------------------------------------------------------------

const (
	// ServerRunTurn asks the daemon to run one turn against one CLI.
	ServerRunTurn = "run_turn"
	// ServerProbe asks the daemon to re-report what is installed.
	ServerProbe = "probe"
)

type ServerMessage struct {
	Type string   `json:"type"`
	Turn *RunTurn `json:"turn,omitempty"`
}

// ReplayEntry is one historical message being handed to a provider that has not
// seen it. Sent only when a catch-up is actually being paid for.
type ReplayEntry struct {
	Role     string `json:"role"`
	Provider string `json:"provider,omitempty"`
	Text     string `json:"text"`
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
)

type DaemonMessage struct {
	Type string `json:"type"`

	// hello
	Machine   string     `json:"machine,omitempty"`
	Providers []Provider `json:"providers,omitempty"`

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
	EventDaemon = "daemon"
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
