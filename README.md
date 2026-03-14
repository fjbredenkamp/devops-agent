# DevOps Agent

An agentic DevOps assistant powered by Claude Opus 4.5, written in Go.

The agent runs an **agentic loop**: it receives your request, decides which
tools to call (shell, file I/O, HTTP probes, git), executes them, and keeps
iterating until it has a complete answer — just like a skilled engineer
working through a problem step by step.

---

## Project Layout

```
devops-agent/
├── cmd/
│   └── agent/
│       └── main.go          # CLI entry point
├── internal/
│   ├── agent/
│   │   └── agent.go         # Core agentic loop (the brain)
│   ├── config/
│   │   └── config.go        # Config from environment variables
│   └── tools/
│       └── tools.go         # Tool registry + all tool handlers
├── pkg/
│   ├── executor/
│   │   └── executor.go      # Shell command runner
│   ├── git/
│   │   └── git.go           # Git status and log
│   ├── health/
│   │   └── health.go        # HTTP health probing
│   └── k8s/
│       └── k8s.go           # Kubernetes (stub — extend me!)
├── .env.example             # Copy to .env and fill in
├── go.mod
└── Makefile
```

---

## Quick Start

### Prerequisites

- Go 1.22+ — [go.dev/dl](https://go.dev/dl)
- An Anthropic API key — [console.anthropic.com](https://console.anthropic.com)

### Setup

```bash
# 1. Copy and fill in your API key
cp .env.example .env
# edit .env and set ANTHROPIC_API_KEY=sk-ant-...

# 2. Load the env and fetch dependencies
source .env
make tidy

# 3. Run!
make run
```

---

## Available Tools

| Tool         | What it does                                    |
|--------------|-------------------------------------------------|
| `shell`      | Run any shell command, returns stdout+stderr     |
| `read_file`  | Read a file's contents                          |
| `write_file` | Create or update a file                         |
| `http_probe` | Check if an HTTP endpoint is up + latency       |
| `git_info`   | Branch, status, and recent commits of a repo    |

---

## Example Prompts

```
You › check if https://api.github.com is healthy

You › show me the last 5 git commits in /path/to/my/repo

You › what processes are listening on port 8080?

You › read my docker-compose.yml and tell me what services are defined

You › create a health check script that probes these three endpoints: ...

You › my CI pipeline failed — here's the log: [paste log]. What went wrong?
```

---

## How the Agentic Loop Works

```
User message
     │
     ▼
┌─────────────────────────────┐
│     Claude Opus 4.5         │
│  Reasons about the task     │
│  Decides which tool to call │
└────────────────┬────────────┘
                 │ tool_use
                 ▼
┌─────────────────────────────┐
│     Tool Registry           │
│  Executes: shell / file /   │
│  http_probe / git_info      │
└────────────────┬────────────┘
                 │ tool_result
                 ▼
┌─────────────────────────────┐
│     Claude Opus 4.5         │
│  Reads result, continues    │
│  May call more tools or     │
│  return final answer        │
└─────────────────────────────┘
```

---

## Extending the Agent

### Adding a new tool

1. **Add a handler** in `internal/tools/tools.go`:
   ```go
   r.register("my_tool", r.myToolHandler)

   func (r *Registry) myToolHandler(ctx context.Context, input json.RawMessage) ToolResult {
       // parse params, do work, return ToolResult{Output: "..."}
   }
   ```

2. **Add a schema** to `AllSchemas()` so the model knows how to call it:
   ```go
   {
       Name:        "my_tool",
       Description: "What this tool does (the model reads this!)",
       InputSchema: map[string]interface{}{...},
   }
   ```

That's it — the agent will automatically start using it.

### Adding Kubernetes support

1. `go get k8s.io/client-go@latest`
2. Implement functions in `pkg/k8s/k8s.go`
3. Wire them into `internal/tools/tools.go`

---

## Learning Resources

- [Go Tour](https://go.dev/tour) — learn Go interactively (free, ~4 hours)
- [Go by Example](https://gobyexample.com) — idiomatic Go patterns
- [Anthropic API Docs](https://docs.anthropic.com) — tool use, models, etc.
- [Anthropic Go SDK](https://github.com/anthropics/anthropic-sdk-go) — the SDK used here
