package agent

import (
	"os"
	"strings"
	"testing"
)

// B-52: the claude CLI's list_models reply, captured verbatim from a signed-in
// 2.1.280 on 2026-09-23, is read as the list the app offers. Before this the
// rows were read one level too high and under the wrong names, so every answer
// looked empty and the app sat on a static list from September.
func TestClaudeModelsComeFromTheCLI(t *testing.T) {
	out, err := os.ReadFile("testdata/claude-list-models.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	models, everyday, ok := parseClaudeModels(out)
	if !ok {
		t.Fatal("a successful reply was not read")
	}

	var got []string
	for _, m := range models {
		got = append(got, m.ID+"="+m.Label)
	}
	want := []string{
		"claude-opus-5-5=Opus 5.5",
		"claude-sonnet-5=Sonnet 5",
		"claude-fable-5-1=Fable 5.1",
		"claude-haiku-4-5-20251001=Haiku 4.5",
		"claude-opus-5=Opus 5",
		"claude-fable-5=Fable 5",
		"claude-opus-4-8=Opus 4.8",
		"claude-opus-4-7=Opus 4.7",
		"claude-opus-4-6=Opus 4.6",
		"claude-sonnet-4-6=Sonnet 4.6",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("models:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// "Default (recommended)" points at Opus 5.5 and is not a model of its own.
	// A new conversation starts on whatever "sonnet" means today.
	if everyday != "claude-sonnet-5" {
		t.Errorf("a new conversation starts on %q, want the sonnet alias's model", everyday)
	}
}

// B-54: the picker Anthropic moved the owner's account to on 2026-09-23 names
// families ("Opus") and puts the version in the description. The app has to
// say which Opus — and run the 1M-context Fable the picker actually offers.
// Captured verbatim from the owner's CLI, signed in the way his daemon is.
func TestAFamilyNamedRowIsLabelledWithItsVersion(t *testing.T) {
	out, err := os.ReadFile("testdata/claude-list-models-families.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	models, everyday, ok := parseClaudeModels(out)
	if !ok {
		t.Fatal("a successful reply was not read")
	}
	var got []string
	for _, m := range models {
		got = append(got, m.ID+"="+m.Label)
	}
	want := []string{
		"claude-sonnet-5=Sonnet 5",
		"claude-fable-5-1[1m]=Fable 5.1",
		"claude-opus-5-5=Opus 5.5",
		"claude-haiku-4-5-20251001=Haiku 4.5",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("models:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if everyday != "claude-sonnet-5" {
		t.Errorf("a new conversation starts on %q, want claude-sonnet-5", everyday)
	}
}

// A name that already carries its version is kept, and a description that is
// about something else is never promoted to a label.
func TestALabelIsTakenFromTheDescriptionOnlyWhenItNamesTheSameModel(t *testing.T) {
	for _, c := range []struct{ name, desc, want string }{
		{"Opus 5.5", "Best for everyday, complex tasks", "Opus 5.5"},
		{"Opus", "Opus 5.5 · Best for everyday, complex tasks", "Opus 5.5"},
		{"Opus", "Best for everyday, complex tasks", "Opus"},
		{"Opus", "Sonnet 5 · for comparison", "Opus"},
		{"Default (recommended)", "Sonnet 5 · Efficient for routine tasks", "Default (recommended)"},
		{"", "", "claude-x"},
	} {
		if got := claudeModelLabel(claudeModelInfo{DisplayName: c.name, Description: c.desc}, "claude-x"); got != c.want {
			t.Errorf("%q / %q labelled %q, want %q", c.name, c.desc, got, c.want)
		}
	}
}

// Two tokens for one model show once. A signed-out CLI answers with its
// built-in defaults, in which a legacy "Opus 4.1" row resolves to Opus 5.
func TestAModelNamedTwiceIsOfferedOnce(t *testing.T) {
	out, err := os.ReadFile("testdata/claude-list-models-signed-out.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	models, _, ok := parseClaudeModels(out)
	if !ok {
		t.Fatal("a successful reply was not read")
	}
	seen := map[string]string{}
	for _, m := range models {
		if prev, dup := seen[m.ID]; dup {
			t.Errorf("%s offered twice, as %q and %q", m.ID, prev, m.Label)
		}
		seen[m.ID] = m.Label
	}
	if seen["claude-opus-5"] != "Opus 5" {
		t.Errorf("claude-opus-5 is labelled %q, want the row that names it rather than a legacy alias", seen["claude-opus-5"])
	}
}

func TestAnUnusableReplyFallsBack(t *testing.T) {
	for name, out := range map[string]string{
		// What a CLI older than list_models says, and exits 0 after.
		"unsupported": `{"type":"control_response","response":{"subtype":"error","request_id":"sg-list-models","error":"Unsupported control request subtype: list_models"}}`,
		"no reply":    `{"type":"system","subtype":"init"}`,
		"someone else's reply": `{"type":"control_response","response":{"subtype":"success","request_id":"other",` +
			`"response":{"models":[{"value":"opus","resolvedModel":"claude-opus-5-5","displayName":"Opus 5.5"}]}}}`,
		// Greyed out in the CLI's own picker ("update to use it"): nothing
		// here can run it, so a list of only those is no list.
		"only disabled": `{"type":"control_response","response":{"subtype":"success","request_id":"sg-list-models",` +
			`"response":{"models":[{"value":"claude-fable-6","resolvedModel":"claude-fable-6","displayName":"Fable 6","disabled":true}]}}}`,
	} {
		if models, _, ok := parseClaudeModels([]byte(out)); ok {
			t.Errorf("%s: read as %v, want a fallback", name, models)
		}
	}
}
