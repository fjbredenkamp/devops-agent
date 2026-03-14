// Package git provides utilities for inspecting git repositories.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Info returns a formatted summary of the git repository at repoPath,
// including current branch, status, and the last N commit messages.
func Info(ctx context.Context, repoPath string, logLines int) (string, error) {
	var sb strings.Builder

	// Current branch
	branch, err := git(ctx, repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("not a git repository or git not installed: %w", err)
	}
	sb.WriteString(fmt.Sprintf("Branch: %s\n\n", branch))

	// Short status
	status, _ := git(ctx, repoPath, "status", "--short")
	if status == "" {
		status = "(clean)"
	}
	sb.WriteString(fmt.Sprintf("Status:\n%s\n\n", status))

	// Recent log
	logOut, _ := git(ctx, repoPath,
		"log",
		fmt.Sprintf("--max-count=%d", logLines),
		"--pretty=format:%h  %as  %an  %s",
	)
	sb.WriteString(fmt.Sprintf("Recent commits (last %d):\n%s\n", logLines, logOut))

	return sb.String(), nil
}

// git runs a git subcommand in the specified directory and returns trimmed output.
func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}
