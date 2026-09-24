package agent

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// B-32: the prompt never rides on the command line, whatever its length.
func TestAgyArgsCarryNoPromptAndReplaceTheFiveMinuteLimit(t *testing.T) {
	args := agyArgs(ExecOptions{Model: "gemini-3.8-flash-high", ResumeSessionID: "conv-1"}, `C:\tmp\agy.log`)
	joined := strings.Join(args, " ")
	for _, want := range []string{"-p=", "--input-format stream-json", "--output-format stream-json",
		"--print-timeout 24h0m0s", `--log-file C:\tmp\agy.log`, "--model gemini-3.8-flash-high", "--conversation conv-1"} {
		if !strings.Contains(joined, want) {
			t.Errorf("agy args missing %q: %v", want, args)
		}
	}
	// A bare -p would take the next flag as the prompt.
	if contains(args, "-p") {
		t.Errorf("-p must carry its value attached: %v", args)
	}
}

// B-55: agy works in the conversation's folder, not in its own scratch one.
func TestAgyIsGivenTheConversationsFolder(t *testing.T) {
	args := strings.Join(agyArgs(ExecOptions{Cwd: `D:\sparstrowgen`}, ""), " ")
	if !strings.Contains(args, `--add-dir D:\sparstrowgen`) {
		t.Errorf("agy is not given the conversation's folder: %s", args)
	}
	if args := agyArgs(ExecOptions{}, ""); contains(args, "--add-dir") {
		t.Errorf("no folder, and still --add-dir: %v", args)
	}
}

func TestAgyInputIsOneUserEvent(t *testing.T) {
	long := strings.Repeat("x", 40_000)
	data, err := agyInput(long)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "\n") || strings.Count(string(data), "\n") != 1 {
		t.Fatalf("want exactly one line, got %d newlines", strings.Count(string(data), "\n"))
	}
	var msg struct {
		Event   string `json:"event"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Event != "user" || msg.Message.Role != "user" || msg.Message.Content != long {
		t.Errorf("event %q role %q, content kept: %v", msg.Event, msg.Message.Role, msg.Message.Content == long)
	}
}

func TestClaudeInputIsOneUserMessage(t *testing.T) {
	data, err := claudeInput("line one\nline two")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "\n") != 1 {
		t.Fatalf("an embedded newline must be escaped, not split the message: %q", data)
	}
	var msg struct {
		Type    string `json:"type"`
		Message struct {
			Content []struct{ Type, Text string } `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Type != "user" || len(msg.Message.Content) != 1 || msg.Message.Content[0].Text != "line one\nline two" {
		t.Errorf("claude input = %s", data)
	}
}

func TestAnAgyModelThisComputerDoesNotListIsRefused(t *testing.T) {
	catalog := []protocol.Model{{ID: "gemini-3.8-flash-high"}, {ID: "gemini-3.8-flash-low"}}
	if err := agyModelError("gemini-3.8-flash-high", catalog); err != nil {
		t.Errorf("a listed model was refused: %v", err)
	}
	err := agyModelError("gemini-9-imaginary", catalog)
	if err == nil || !strings.Contains(err.Error(), "gemini-9-imaginary") || !strings.Contains(err.Error(), "gemini-3.8-flash-low") {
		t.Errorf("an unlisted model should be refused naming it and the choices: %v", err)
	}
	// No catalogue, or no model chosen: agy decides.
	if agyModelError("anything", nil) != nil || agyModelError("", catalog) != nil {
		t.Error("with no catalogue or no model the turn must not be blocked")
	}
}

// Captured 2026-09-14 from agy 1.2.3 asked to read a file without permission.
func TestAnAgyTurnRefusedAToolIsAFailureNotABlankAnswer(t *testing.T) {
	const stream = `{"event":"result","result":{"conversation_id":"1f561bd0","status":"SUCCESS","response":"","usage":{"total_tokens":26467},"denied_actions":[{"action":"command","display_name":"RunCommand"}]}}`
	p, _ := drain(t, func(ch chan<- Message) parsed {
		return parseAgy(strings.NewReader(stream), ch)
	})
	if p.Err == nil || !strings.Contains(p.Err.Error(), "RunCommand") {
		t.Fatalf("err = %v, want a failure naming the refused tool", p.Err)
	}
}

func TestAgyFailedStatusCarriesItsOwnReason(t *testing.T) {
	const stream = `{"event":"result","result":{"status":"ERROR","response":"","error":"failed to decode stream input"}}`
	p, _ := drain(t, func(ch chan<- Message) parsed {
		return parseAgy(strings.NewReader(stream), ch)
	})
	if p.Err == nil || !strings.Contains(p.Err.Error(), "failed to decode stream input") {
		t.Fatalf("err = %v", p.Err)
	}
}

func TestAnEmptyAgyAnswerIsExplainedFromItsLog(t *testing.T) {
	cases := []struct {
		name, log string
		wait      error
		want      string
	}{
		{"provider error", "I0914 printmode.go] agent executor error: model overloaded\n", nil, "model overloaded"},
		{"its own timeout", "E0914 printmode.go:289] Print mode: timed out after 100 polls (printed=3)\n", nil, "gave up"},
		{"bad exit", "", errors.New("exit status 1"), "exit status 1"},
		{"nothing at all", "", nil, "without writing an answer"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := agyQuietFailure(c.log, c.wait); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want it to say %q", err, c.want)
			}
		})
	}
}
