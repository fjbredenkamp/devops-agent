// Package tools defines all tool schemas and their handlers for the DevOps agent.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yourname/devops-agent/pkg/executor"
	"github.com/yourname/devops-agent/pkg/git"
	"github.com/yourname/devops-agent/pkg/health"
)

// ToolResult is returned after a tool call is executed.
type ToolResult struct {
	Output string
	IsError bool
}

// Handler is a function that executes a tool given its JSON input.
type Handler func(ctx context.Context, input json.RawMessage) ToolResult

// Registry maps tool names to their handlers.
type Registry struct {
	handlers   map[string]Handler
	workingDir string
}

// New creates a Registry with all built-in DevOps tools registered.
func New(workingDir string) *Registry {
	r := &Registry{
		handlers:   make(map[string]Handler),
		workingDir: workingDir,
	}
	r.register("shell", r.shellHandler)
	r.register("read_file", r.readFileHandler)
	r.register("write_file", r.writeFileHandler)
	r.register("http_probe", r.httpProbeHandler)
	r.register("git_info", r.gitInfoHandler)
	return r
}

func (r *Registry) register(name string, h Handler) {
	r.handlers[name] = h
}

// Execute dispatches a tool call by name.
func (r *Registry) Execute(ctx context.Context, name string, input json.RawMessage) ToolResult {
	h, ok := r.handlers[name]
	if !ok {
		return ToolResult{
			Output:  fmt.Sprintf("unknown tool: %q", name),
			IsError: true,
		}
	}
	return h(ctx, input)
}

// ─── Tool Handlers ────────────────────────────────────────────────────────────

// shellHandler runs a shell command and returns combined stdout+stderr.
func (r *Registry) shellHandler(ctx context.Context, input json.RawMessage) ToolResult {
	var params struct {
		Command    string `json:"command"`
		WorkingDir string `json:"working_dir"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return ToolResult{Output: "invalid params: " + err.Error(), IsError: true}
	}
	dir := params.WorkingDir
	if dir == "" {
		dir = r.workingDir
	}
	out, err := executor.Run(ctx, params.Command, dir)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("exit error: %v\n%s", err, out), IsError: true}
	}
	return ToolResult{Output: out}
}

// readFileHandler reads a file and returns its contents.
func (r *Registry) readFileHandler(_ context.Context, input json.RawMessage) ToolResult {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return ToolResult{Output: "invalid params: " + err.Error(), IsError: true}
	}
	absPath := params.Path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(r.workingDir, absPath)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("read error: %v", err), IsError: true}
	}
	return ToolResult{Output: string(data)}
}

// writeFileHandler writes content to a file, creating directories as needed.
func (r *Registry) writeFileHandler(_ context.Context, input json.RawMessage) ToolResult {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return ToolResult{Output: "invalid params: " + err.Error(), IsError: true}
	}
	absPath := params.Path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(r.workingDir, absPath)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return ToolResult{Output: fmt.Sprintf("mkdir error: %v", err), IsError: true}
	}
	if err := os.WriteFile(absPath, []byte(params.Content), 0644); err != nil {
		return ToolResult{Output: fmt.Sprintf("write error: %v", err), IsError: true}
	}
	return ToolResult{Output: fmt.Sprintf("wrote %d bytes to %s", len(params.Content), absPath)}
}

// httpProbeHandler checks the health of an HTTP endpoint.
func (r *Registry) httpProbeHandler(ctx context.Context, input json.RawMessage) ToolResult {
	var params struct {
		URL            string `json:"url"`
		TimeoutSeconds int    `json:"timeout_seconds"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return ToolResult{Output: "invalid params: " + err.Error(), IsError: true}
	}
	if params.TimeoutSeconds == 0 {
		params.TimeoutSeconds = 10
	}
	result, err := health.Probe(ctx, params.URL, params.TimeoutSeconds)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("probe error: %v", err), IsError: true}
	}
	return ToolResult{Output: result}
}

// gitInfoHandler returns git status/log for a repository path.
func (r *Registry) gitInfoHandler(ctx context.Context, input json.RawMessage) ToolResult {
	var params struct {
		RepoPath string `json:"repo_path"`
		LogLines int    `json:"log_lines"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return ToolResult{Output: "invalid params: " + err.Error(), IsError: true}
	}
	if params.RepoPath == "" {
		params.RepoPath = r.workingDir
	}
	if params.LogLines == 0 {
		params.LogLines = 10
	}
	result, err := git.Info(ctx, params.RepoPath, params.LogLines)
	if err != nil {
		return ToolResult{Output: fmt.Sprintf("git error: %v", err), IsError: true}
	}
	return ToolResult{Output: result}
}

// ─── Tool Schemas (sent to the API) ──────────────────────────────────────────

// Schema describes one tool to the Anthropic API.
type Schema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// AllSchemas returns the JSON tool definitions for all registered tools.
func AllSchemas() []Schema {
	return []Schema{
		{
			Name:        "shell",
			Description: "Run a shell command on the local machine. Returns combined stdout and stderr. Use for running scripts, checking services, inspecting processes, etc.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command":     map[string]string{"type": "string", "description": "The shell command to execute"},
					"working_dir": map[string]string{"type": "string", "description": "Working directory (optional, defaults to agent working dir)"},
				},
				"required": []string{"command"},
			},
		},
		{
			Name:        "read_file",
			Description: "Read the contents of a file. Useful for inspecting configs, logs, Dockerfiles, CI/CD pipeline definitions, etc.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]string{"type": "string", "description": "Path to the file (absolute or relative to working dir)"},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "write_file",
			Description: "Write content to a file. Use to create or update configs, scripts, manifests, or any other file. Creates parent directories automatically.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path":    map[string]string{"type": "string", "description": "Destination file path"},
					"content": map[string]string{"type": "string", "description": "Content to write to the file"},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			Name:        "http_probe",
			Description: "Check the availability of an HTTP/HTTPS endpoint. Returns status code, latency, and whether the service is healthy.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url":             map[string]string{"type": "string", "description": "The URL to probe"},
					"timeout_seconds": map[string]string{"type": "integer", "description": "Request timeout in seconds (default: 10)"},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        "git_info",
			Description: "Get git status and recent commit log for a repository. Useful for understanding recent changes, branch status, and uncommitted modifications.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_path": map[string]string{"type": "string", "description": "Path to the git repository (defaults to working dir)"},
					"log_lines": map[string]string{"type": "integer", "description": "Number of recent commits to include (default: 10)"},
				},
				"required": []string{},
			},
		},
	}
}
