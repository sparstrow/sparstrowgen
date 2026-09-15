//go:build !windows

package agent

// readUserEnvironment has nothing to add outside Windows, where a user's
// variables live in their shell profile rather than in one readable place.
func readUserEnvironment() map[string]string { return nil }
