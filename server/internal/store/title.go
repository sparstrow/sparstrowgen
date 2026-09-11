package store

import (
	"strings"
	"unicode"
)

// titleLimit is a title's length in runes. Long enough for a sentence's worth of
// subject — "why does the daemon drop its socket when the laptop sleeps" is 58
// — and short enough that the sidebar truncates for width rather than for this.
const titleLimit = 60

/*
	Naming a conversation from the first thing said in it.

The alternative was a model call per conversation, which names them better and
costs a request, a failure mode and a wait. This costs nothing and is right most
of the time, because the first message of a coding conversation is nearly always
a statement of the task. When it is wrong the owner renames it, which he could
always do — so the worst case is the "Untitled conversation" we have now, with
extra steps.

What it must not do is produce something worse than no name: a row of backticks,
a fence marker, half a URL. So it skips what is punctuation rather than words and
gives up rather than inventing, and an empty result leaves the conversation
unnamed for the placeholder to describe.
*/
func titleFrom(text string) string {
	line := firstMeaningfulLine(text)
	if line == "" {
		return ""
	}
	return truncate(strings.Join(strings.Fields(line), " "), titleLimit)
}

// firstMeaningfulLine finds the first line carrying a word, with its markdown
// lead-in removed. A pasted message often opens with a fence or a heading, and
// neither is what the message is about.
func firstMeaningfulLine(text string) string {
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
			continue
		}
		line = stripMarkers(line)
		if hasWord(line) {
			return line
		}
	}
	return ""
}

// stripMarkers peels off the prefixes that mark a line's kind rather than say
// anything — repeatedly, because they nest: "> - **fix the thing**".
func stripMarkers(line string) string {
	for {
		before := line
		line = strings.TrimLeft(line, "#>*-+_` \t")
		if rest, ok := afterOrderedMarker(line); ok {
			line = rest
		}
		line = strings.TrimRight(line, "*_` \t")
		if line == before {
			return line
		}
	}
}

// afterOrderedMarker recognises "1." and "2)" at the head of a list item.
func afterOrderedMarker(line string) (string, bool) {
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(line) || (line[i] != '.' && line[i] != ')') {
		return line, false
	}
	return strings.TrimLeft(line[i+1:], " \t"), true
}

func hasWord(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// truncate cuts at a word boundary where there is one. A hard cut mid-word reads
// as a bug rather than as a shortening, and one long token — a path, a URL — has
// no boundary to find, so it is cut where it must be.
func truncate(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	cut := string(runes[:limit])
	if at := strings.LastIndexAny(cut, " \t"); at > limit/2 {
		cut = cut[:at]
	}
	return strings.TrimRight(cut, " \t.,;:") + "…"
}
