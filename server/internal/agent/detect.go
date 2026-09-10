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
// Model lists come from the CLI wherever the CLI can answer. Only agy can
// enumerate directly; claude has a list_models control request that our 2.1.90
// is too old for; codex has nothing. Each fallback below is a captured fact,
// not a remembered one — see docs/KnownGaps.md G-8.
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
	models := claudeModels(ctx)
	p := protocol.Provider{
		ID:           "claude",
		Label:        "claude",
		Models:       models,
		Availability: protocol.Available,
		// The only one of the three that states a real dollar figure.
		ReportsUsd: true,
		// No capture has produced a text delta yet, so the surface must not
		// promise one. The adapter still reads them if they appear.
		Streams: false,
		Routes:  false,
	}
	if len(models) > 0 {
		p.Model = &models[1] // Sonnet: the everyday default, not the expensive one
	}
	return p
}

// claudeStaticModels is the fallback catalogue. Each id was resolved by running
// the alias and reading `model` back out of the system.init line, not from
// memory — an earlier version of this list said "Opus 5", which this CLI does
// not offer.
var claudeStaticModels = []protocol.Model{
	{ID: "claude-opus-4-6", Label: "Opus 4.6"},
	{ID: "claude-sonnet-4-6", Label: "Sonnet 4.6"},
	{ID: "claude-haiku-4-5-20251001", Label: "Haiku 4.5"},
}

// claudeModels asks the CLI first and falls back to the static catalogue.
//
// The control request sends no user message, so nothing is billed. An old CLI
// answers "Unsupported control request subtype" in about two seconds and exits
// 0 — it does not hang — which is why this needs no version gate. Borrowed from
// Multica's server/pkg/agent/claude_models.go.
func claudeModels(ctx context.Context) []protocol.Model {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	cmd := command(ctx, "", "claude",
		"--print", "--verbose",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--strict-mcp-config",
	)
	cmd.Stdin = strings.NewReader(
		`{"type":"control_request","request_id":"sg","request":{"subtype":"list_models"}}` + "\n")

	out, err := cmd.Output()
	if err != nil {
		return claudeStaticModels
	}

	for _, line := range strings.Split(string(out), "\n") {
		var resp struct {
			Type     string `json:"type"`
			Response struct {
				Subtype string `json:"subtype"`
				Models  []struct {
					Model       string `json:"model"`
					DisplayName string `json:"display_name"`
				} `json:"models"`
			} `json:"response"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(line)), &resp) != nil {
			continue
		}
		if resp.Type != "control_response" || len(resp.Response.Models) == 0 {
			continue
		}
		models := make([]protocol.Model, 0, len(resp.Response.Models))
		for _, m := range resp.Response.Models {
			label := m.DisplayName
			if label == "" {
				label = m.Model
			}
			models = append(models, protocol.Model{ID: m.Model, Label: label})
		}
		return models
	}
	return claudeStaticModels
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
