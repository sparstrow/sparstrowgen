package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// Detect reports what is actually installed and usable on this machine.
//
// Model lists come from the CLI wherever the CLI can answer: agy enumerates
// directly, claude answers a list_models control request, and codex has
// nothing. Each fallback below is a captured fact, not a remembered one — see
// docs/KnownGaps.md G-8.
func Detect(ctx context.Context) []protocol.Provider {
	return []protocol.Provider{
		detectClaude(ctx),
		detectCodex(ctx),
		detectAgy(ctx),
	}
}

func missing(id string) protocol.Provider {
	return protocol.Provider{
		ID:                id,
		Label:             id,
		Models:            []protocol.Model{},
		Availability:      protocol.Blocked,
		UnavailableReason: "Not installed",
	}
}

// ---------------------------------------------------------------------------

func detectClaude(ctx context.Context) protocol.Provider {
	if _, err := exec.LookPath("claude"); err != nil {
		return missing("claude")
	}
	models, everyday := claudeModels(ctx)
	p := protocol.Provider{
		ID:           "claude",
		Label:        "claude",
		Models:       models,
		Availability: protocol.Available,
		// The only one of the three that states a real dollar figure.
		ReportsUsd: true,
		// And the only one that emits rate_limit_event. It arrives mid-turn, so
		// there is nothing to show until a message has been sent.
		ReportsLimits: true,
		// VERIFIED 2026-09-10 once auth worked: 89 content_block_delta events
		// averaging 8.1 characters for a four-sentence answer — finer-grained
		// than agy. Needs --include-partial-messages, which is in the args.
		Streams: true,
		Routes:  false,
	}
	for i := range models {
		if models[i].ID == everyday {
			p.Model = &models[i]
		}
	}
	if p.Model == nil && len(models) > 0 {
		p.Model = &models[0]
	}
	return p
}

// claudeStaticModels is the fallback catalogue, used only when the CLI cannot
// answer list_models. It is the head of each family as a signed-in 2.1.280
// reported it on 2026-09-23 — and it goes stale the day the next model ships,
// which is exactly why it is no longer the first answer.
var claudeStaticModels = []protocol.Model{
	{ID: "claude-opus-5-5", Label: "Opus 5.5"},
	{ID: "claude-sonnet-5", Label: "Sonnet 5"},
	{ID: "claude-haiku-4-5-20251001", Label: "Haiku 4.5"},
}

// claudeStaticEveryday is what a new conversation starts on when the CLI
// could not be asked.
const claudeStaticEveryday = "claude-sonnet-5"

// claudeListModelsID labels our control request so its reply can be picked out
// of the stream.
const claudeListModelsID = "sg-list-models"

// claudeModels asks the CLI for its catalogue and falls back to the static one.
// It also returns the model a new conversation should start on.
//
// The answer is the list the CLI's own /model picker shows, computed by the
// installed binary against the signed-in account, so a model Anthropic ships
// appears here with nothing in this repo changing (docs/Bugs.md B-52). The
// request sends no user message, so nothing is billed, and a CLI too old to
// know it answers "Unsupported control request subtype" and exits rather than
// hanging. Borrowed from Multica's server/pkg/agent/claude_models.go.
func claudeModels(ctx context.Context) ([]protocol.Model, string) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	cmd := command(ctx, "", "claude",
		"--print", "--verbose",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		// Without an --mcp-config this means no MCP servers at all: listing
		// models has no use for them, and booting them could make it slow.
		"--strict-mcp-config",
	)
	cmd.Stdin = strings.NewReader(
		`{"type":"control_request","request_id":"` + claudeListModelsID + `","request":{"subtype":"list_models"}}` + "\n")

	out, err := cmd.Output()
	if err != nil {
		return claudeStaticModels, claudeStaticEveryday
	}
	models, everyday, ok := parseClaudeModels(out)
	if !ok {
		return claudeStaticModels, claudeStaticEveryday
	}
	return models, everyday
}

// claudeModelInfo is one row of the CLI's catalogue.
//
// Value is the picker's token — an alias like "opus", an id, or the sentinel
// "default" — and ResolvedModel is what that token runs today. The resolved id
// is what gets stored and passed to --model: an alias moves when Anthropic
// moves it, and a transcript has to keep saying which model actually answered.
type claudeModelInfo struct {
	Value         string `json:"value"`
	ResolvedModel string `json:"resolvedModel"`
	DisplayName   string `json:"displayName"`
	Disabled      bool   `json:"disabled"`
}

// parseClaudeModels reads the list_models reply out of the CLI's stdout.
//
// The doubled `response` is Claude's shape, not a slip: the outer one is the
// envelope, the inner one the payload. Reading the rows one level too high,
// under the wrong names, is what left the app on a list from September while
// the CLI was offering Opus 5.5 (docs/Bugs.md B-52).
//
// Rows are keyed by the model they run, so two tokens for one model show once.
// "default" is not offered as a model of its own, because it is only a pointer
// to one, and a row the CLI greys out is left out because nothing here can run
// it. ok is false when there is no usable answer, so the caller falls back.
func parseClaudeModels(out []byte) (models []protocol.Model, everyday string, ok bool) {
	for _, line := range strings.Split(string(out), "\n") {
		var resp struct {
			Type     string `json:"type"`
			Response struct {
				Subtype   string `json:"subtype"`
				RequestID string `json:"request_id"`
				Response  struct {
					Models []claudeModelInfo `json:"models"`
				} `json:"response"`
			} `json:"response"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(line)), &resp) != nil {
			continue
		}
		if resp.Type != "control_response" || resp.Response.RequestID != claudeListModelsID {
			continue
		}
		if resp.Response.Subtype != "success" {
			return nil, "", false
		}

		seen := map[string]bool{}
		var recommended, sonnet string
		for _, m := range resp.Response.Response.Models {
			id := strings.TrimSpace(m.ResolvedModel)
			if id == "" {
				id = strings.TrimSpace(m.Value)
			}
			if id == "" || m.Disabled {
				continue
			}
			switch m.Value {
			case "default":
				recommended = id
				continue
			case "sonnet":
				sonnet = id
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			label := strings.TrimSpace(m.DisplayName)
			if label == "" {
				label = id
			}
			models = append(models, protocol.Model{ID: id, Label: label})
		}
		if len(models) == 0 {
			return nil, "", false
		}

		// A new conversation starts on whatever "sonnet" means today: the
		// everyday model rather than the expensive one, a deliberate choice for
		// his quota that is kept — only now it follows Anthropic's alias rather
		// than a position in a hand-kept list. Then the CLI's own
		// recommendation, then the first row.
		switch {
		case seen[sonnet]:
			everyday = sonnet
		case seen[recommended]:
			everyday = recommended
		default:
			everyday = models[0].ID
		}
		return models, everyday, true
	}
	return nil, "", false
}

// ---------------------------------------------------------------------------

// codexStaticModels: codex has no list command. These names were read out of
// the shipped binary and cross-checked against `model = "gpt-5.6-sol"` in
// ~/.codex/config.toml. The valid set is account-dependent — an unsupported
// model returns a 400 naming no alternatives — so this list is a starting
// point, never an authority.
var codexStaticModels = []protocol.Model{
	{ID: "gpt-5.6-sol", Label: "GPT-5.6 Sol"},
	{ID: "gpt-5.6-luna", Label: "GPT-5.6 Luna"},
	{ID: "gpt-5.6-terra", Label: "GPT-5.6 Terra"},
}

func detectCodex(ctx context.Context) protocol.Provider {
	if _, err := exec.LookPath("codex"); err != nil {
		return missing("codex")
	}
	models := codexStaticModels
	p := protocol.Provider{
		ID:           "codex",
		Label:        "codex",
		Models:       models,
		Availability: protocol.Available,
		ReportsUsd:   false,
		// Verified: no delta event type exists in its --json stream.
		Streams: false,
		Routes:  false,
	}
	p.Model = &models[0]
	return p
}

// ---------------------------------------------------------------------------

func detectAgy(ctx context.Context) protocol.Provider {
	if _, err := exec.LookPath("agy"); err != nil {
		return missing("agy")
	}
	models := agyModels(ctx)
	agyCatalog.put(models)
	p := protocol.Provider{
		ID:           "agy",
		Label:        "agy",
		Models:       models,
		Availability: protocol.Available,
		ReportsUsd:   false,
		Streams:      true,
		// A router: Gemini, Claude and GPT-OSS models behind one CLI.
		Routes: true,
	}
	if len(models) > 0 {
		p.Model = &models[0]
	}
	if len(models) == 0 {
		p.Availability = protocol.Blocked
		p.UnavailableReason = "No models available"
	}
	return p
}

// agyModels runs `agy models`, which prints "<id>\t<label>" per line. It is the
// only one of the three CLIs that enumerates.
func agyModels(ctx context.Context) []protocol.Model {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	out, err := command(ctx, "", "agy", "models").Output()
	if err != nil {
		return []protocol.Model{}
	}
	models := []protocol.Model{}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		id, label, ok := strings.Cut(strings.TrimSpace(scanner.Text()), "\t")
		if !ok || id == "" {
			continue // the "Fetching available models..." preamble
		}
		models = append(models, protocol.Model{ID: id, Label: strings.TrimSpace(label)})
	}
	return models
}

// Backends returns the driver for each provider id.
func Backends() map[string]Backend {
	return map[string]Backend{
		"claude": Claude{},
		"codex":  Codex{},
		"agy":    Agy{},
	}
}
