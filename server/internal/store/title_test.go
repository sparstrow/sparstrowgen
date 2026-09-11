package store

import "testing"

/* What a conversation gets called when nobody has said.

These are the shapes real first messages actually arrive in — a plain request, a
pasted error, a numbered list, a message that opens with a code fence — and the
bar is not "a good title" but "better than a column of identical rows, and never
worse than no name at all". */
func TestNamingAConversationFromItsFirstMessage(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    string
	}{{
		name:    "the ordinary case, which is most of them",
		message: "Add a stop button to the composer",
		want:    "Add a stop button to the composer",
	}, {
		name:    "only the first line: the rest is detail",
		message: "Fix the folder picker\n\nIt keeps the old folder after a move.",
		want:    "Fix the folder picker",
	}, {
		name:    "a heading is a marker, not the subject",
		message: "## Why does the daemon drop its socket?",
		want:    "Why does the daemon drop its socket?",
	}, {
		name:    "so is a bullet, and so is the emphasis inside it",
		message: "- **rewrite the watchdog**",
		want:    "rewrite the watchdog",
	}, {
		name:    "a numbered list starts at its first item",
		message: "1. Stop the turn\n2. Then retry it",
		want:    "Stop the turn",
	}, {
		name:    "a message that opens with a fence names itself from the code",
		message: "```\npanic: send on closed channel\n```",
		want:    "panic: send on closed channel",
	}, {
		name:    "leading blank lines are not a name",
		message: "\n\n   \nwhy is this failing on linux only",
		want:    "why is this failing on linux only",
	}, {
		name:    "whitespace inside the line is collapsed, not preserved",
		message: "make   the   sidebar\tscannable",
		want:    "make the sidebar scannable",
	}, {
		name:    "a long message is cut at a word, never mid-word",
		message: "The daemon keeps the process tree in a job object so that everything the agent spawned dies with it",
		want:    "The daemon keeps the process tree in a job object so that…",
	}, {
		name:    "one long token has no word to cut at, so it is cut where it must be",
		message: `D:\sparstrowgen\server\internal\agent\tree_windows_test.go:tree_windows_test.go`,
		want:    `D:\sparstrowgen\server\internal\agent\tree_windows_test.go:t…`,
	}, {
		name:    "nothing but punctuation is not a name — better unnamed",
		message: "---\n***\n",
		want:    "",
	}, {
		name:    "and neither is an empty message",
		message: "   \n\t\n",
		want:    "",
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := titleFrom(c.message); got != c.want {
				t.Errorf("titleFrom(%q)\n got %q\nwant %q", c.message, got, c.want)
			}
		})
	}
}

// A name has to fit a sidebar row, and the ellipsis is part of the budget rather
// than something added past it.
func TestANameNeverOutgrowsTheRowItGoesIn(t *testing.T) {
	long := ""
	for len(long) < 400 {
		long += "sparstrowgen "
	}
	got := []rune(titleFrom(long))
	if len(got) > titleLimit+1 {
		t.Errorf("a %d-rune name was produced for a limit of %d", len(got), titleLimit)
	}
}
