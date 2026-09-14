package protocol

// Keeping the installed daemon updated (spec US3).

// DaemonProtocol is the version of this protocol the daemon in this module
// speaks. It rises only when a change would break an older daemon; a daemon
// that reports none is from before it existed and counts as 0.
const DaemonProtocol = 1

// MinDaemonProtocol is the oldest daemon protocol the server still works with.
// A computer below it is told it is too old instead of being sent work. A
// variable so tests can raise it; nothing else changes it.
//
// Raise it only after the daemons below it have had a release they could
// update to (docs/runbooks/release-workflow.md, expand/contract).
var MinDaemonProtocol = 0

const (
	// ServerCheckUpdate asks the daemon to check for a new version now,
	// whatever its automatic setting. Answered with DaemonUpdateStatus echoing
	// the RequestID.
	ServerCheckUpdate = "check_update"
	// ServerApplyUpdate asks the daemon to install the newest version, waiting
	// for running agent work to finish first. Answered like ServerCheckUpdate.
	ServerApplyUpdate = "apply_update"
	// ServerUpdatePreference tells the daemon whether it may install updates by
	// itself. Sent on connect and whenever the person changes it.
	ServerUpdatePreference = "update_preference"
)

// DaemonUpdateStatus reports where an update stands. Sent as an answer to a
// check or apply request, and by itself whenever the status changes.
const DaemonUpdateStatus = "update_status"

// What an UpdateStatus says, as a value rather than a sentence.
const (
	UpdateUnchecked = "unchecked"
	UpdateCurrent   = "current"
	UpdateAvailable = "available"
	// UpdateWaiting: a verified update is ready, and agent work is running.
	UpdateWaiting  = "waiting"
	UpdateUpdating = "updating"
	UpdateFailed   = "failed"
)

type UpdateStatus struct {
	Kind string `json:"kind"`
	// The version on offer, for available, waiting and updating.
	Version string `json:"version,omitempty"`
	// How many agent tasks the update is waiting on.
	ActiveTasks int `json:"activeTasks,omitempty"`
	// For failed: what happened, and what is running now.
	Message string `json:"message,omitempty"`
}
