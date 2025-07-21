package handlers

import (
	"context"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/imjasonh/hooks/internal/hooks"
)

func init() {
	hooks.RegisterHook(hooks.EventPostToolUse, "Write|Edit|MultiEdit", RunGoImportsOnGoFiles)
}

func RunGoImportsOnGoFiles(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	fileInput, err := input.GetFileInput()
	if err != nil {
		slog.Debug("no file input found", "error", err)
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	if fileInput.FilePath == "" {
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	// Check if it's a Go file
	if !strings.HasSuffix(fileInput.FilePath, ".go") {
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	// Check if goimports is available
	if _, err := exec.LookPath("goimports"); err != nil {
		slog.Debug("goimports not found in PATH", "error", err)
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	// Run goimports -w on the file
	cmd := exec.CommandContext(ctx, "goimports", "-w", fileInput.FilePath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		slog.Error("goimports failed",
			"file", fileInput.FilePath,
			"error", err,
			"output", string(output))
		// Don't block on goimports errors
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	if len(output) > 0 {
		slog.Info("goimports applied changes",
			"file", fileInput.FilePath,
			"output", string(output))
	} else {
		slog.Info("goimports completed",
			"file", fileInput.FilePath)
	}

	return hooks.HookOutput{Decision: "approve"}, nil
}
