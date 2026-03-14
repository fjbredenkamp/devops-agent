// Package config loads agent configuration from environment variables.
package config

import (
	"fmt"
	"os"

	anthropic "github.com/anthropics/anthropic-sdk-go"
)

// Config holds all runtime configuration for the agent.
type Config struct {
	AnthropicAPIKey string
	Model           anthropic.Model
	MaxTokens       int64
	WorkingDir      string
}

// Load reads config from environment variables.
// Required: ANTHROPIC_API_KEY
// Optional: AGENT_MODEL, AGENT_WORKING_DIR
func Load() (*Config, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is not set")
	}

	model := anthropic.Model(os.Getenv("AGENT_MODEL"))
	if model == "" {
		model = anthropic.ModelClaudeOpus4_5 // Default to Opus 4.5
	}

	workDir := os.Getenv("AGENT_WORKING_DIR")
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("could not get working directory: %w", err)
		}
	}

	return &Config{
		AnthropicAPIKey: apiKey,
		Model:           model,
		MaxTokens:       4096,
		WorkingDir:      workDir,
	}, nil
}
