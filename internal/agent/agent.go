// Package agent implements the core agentic loop: send message → handle tool calls → repeat.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/yourname/devops-agent/internal/config"
	"github.com/yourname/devops-agent/internal/tools"
)

const systemPrompt = `You are an expert DevOps engineer assistant running on a Linux machine.
You have access to tools that let you inspect and manage the local environment:
  - shell:      run any shell command
  - read_file:  read files (configs, logs, Dockerfiles, etc.)
  - write_file: create or update files
  - http_probe: check if HTTP endpoints are up
  - git_info:   inspect git repositories

Guidelines:
- Always explain what you're about to do before using a tool
- If a command might be destructive, warn the user first
- Prefer targeted, minimal commands over broad ones
- When diagnosing issues, gather information before proposing fixes
- Summarise findings clearly after using tools`

// Agent wraps the Anthropic client and tool registry.
type Agent struct {
	client   anthropic.Client
	cfg      *config.Config
	registry *tools.Registry
	// conversation history persists across Run() calls in a session
	history []anthropic.MessageParam
}

// New creates a new Agent using the provided config.
func New(cfg *config.Config) *Agent {
	client := anthropic.NewClient(option.WithAPIKey(cfg.AnthropicAPIKey))
	return &Agent{
		client:   client,
		cfg:      cfg,
		registry: tools.New(cfg.WorkingDir),
		history:  []anthropic.MessageParam{},
	}
}

// Run sends the user message, executes any tool calls the model requests,
// and returns the final text response. The conversation history is preserved.
func (a *Agent) Run(ctx context.Context, userMessage string) (string, error) {
	// Append user turn to history
	a.history = append(a.history, anthropic.NewUserMessage(
		anthropic.NewTextBlock(userMessage),
	))

	// Build tool definitions for the API
	apiTools := buildAPITools()

	// Agentic loop: keep calling the API until the model stops requesting tools
	for {
		fmt.Print("  [thinking...]")

		resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     a.cfg.Model,
			MaxTokens: a.cfg.MaxTokens,
			System: []anthropic.TextBlockParam{
				{Text: systemPrompt},
			},
			Tools:    apiTools,
			Messages: a.history,
		})
		if err != nil {
			return "", fmt.Errorf("API call failed: %w", err)
		}

		fmt.Print("\r               \r") // clear "thinking..." line

		// Add assistant's response to history
		a.history = append(a.history, resp.ToParam())

		// If the model is done (no more tool use), return the final text
		if resp.StopReason == anthropic.StopReasonEndTurn {
			return extractText(resp.Content), nil
		}

		// Process tool calls
		toolResults, err := a.executeToolCalls(ctx, resp.Content)
		if err != nil {
			return "", err
		}

		// Append tool results as a user turn so the model can continue
		a.history = append(a.history, anthropic.NewUserMessage(toolResults...))
	}
}

// executeToolCalls runs all tool_use blocks in a response and returns result blocks.
func (a *Agent) executeToolCalls(ctx context.Context, content []anthropic.ContentBlockUnion) ([]anthropic.ContentBlockParamUnion, error) {
	var resultBlocks []anthropic.ContentBlockParamUnion

	for _, block := range content {
		switch block.Type {
		case "text":
			tb := block.AsText()
			if tb.Text != "" {
				fmt.Printf("  Agent (reasoning): %s\n", tb.Text)
			}

		case "tool_use":
			b := block.AsToolUse()
			fmt.Printf("  → tool: %s(%s)\n", b.Name, string(b.Input))

			rawInput, err := json.Marshal(b.Input)
			if err != nil {
				return nil, fmt.Errorf("marshaling tool input: %w", err)
			}

			result := a.registry.Execute(ctx, b.Name, rawInput)

			if result.IsError {
				fmt.Printf("  ✗ error: %s\n", truncate(result.Output, 200))
			} else {
				fmt.Printf("  ✓ output: %s\n", truncate(result.Output, 200))
			}

			resultBlocks = append(resultBlocks, anthropic.NewToolResultBlock(b.ID, result.Output, result.IsError))
		}
	}

	return resultBlocks, nil
}

// ResetHistory clears the conversation history (start a new session).
func (a *Agent) ResetHistory() {
	a.history = []anthropic.MessageParam{}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func extractText(content []anthropic.ContentBlockUnion) string {
	var parts []string
	for _, block := range content {
		if block.Type == "text" {
			tb := block.AsText()
			if tb.Text != "" {
				parts = append(parts, tb.Text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func buildAPITools() []anthropic.ToolUnionParam {
	schemas := tools.AllSchemas()
	apiTools := make([]anthropic.ToolUnionParam, len(schemas))
	for i, s := range schemas {
		// Extract required field if present
		var required []string
		if req, ok := s.InputSchema["required"].([]string); ok {
			required = req
		}

		apiTools[i] = anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        s.Name,
				Description: anthropic.String(s.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: s.InputSchema["properties"],
					Required:   required,
				},
			},
		}
	}
	return apiTools
}
