// Package executor runs shell commands safely and returns their output.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Run executes a shell command in the given working directory.
// It combines stdout and stderr so the agent sees the full picture.
// Returns an error if the command exits with a non-zero code, but still
// returns any output that was produced before the failure.
func Run(ctx context.Context, command, workingDir string) (string, error) {
	// We use /bin/sh -c so the command can use pipes, redirects, etc.
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
	cmd.Dir = workingDir

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	output := strings.TrimSpace(buf.String())

	if output == "" && err != nil {
		output = fmt.Sprintf("command failed: %v", err)
	}

	return output, err
}
