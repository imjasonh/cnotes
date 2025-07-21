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
	// Extract file path from parameters
	filePath, ok := input.ToolUseRequest.Parameters["file_path"].(string)
	if !ok {
		return hooks.HookOutput{Decision: "continue"}, nil
	}

	// Check if it's a Go file
	if !strings.HasSuffix(filePath, ".go") {
		return hooks.HookOutput{Decision: "continue"}, nil
	}

	// Check if goimports is available
	if _, err := exec.LookPath("goimports"); err != nil {
		slog.Debug("goimports not found in PATH", "error", err)
		return hooks.HookOutput{Decision: "continue"}, nil
	}

	// Run goimports -w on the file
	cmd := exec.CommandContext(ctx, "goimports", "-w", filePath)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		slog.Error("goimports failed", 
			"file", filePath,
			"error", err,
			"output", string(output))
		// Don't block on goimports errors
		return hooks.HookOutput{Decision: "continue"}, nil
	}

	if len(output) > 0 {
		slog.Info("goimports applied changes",
			"file", filePath,
			"output", string(output))
	} else {
		slog.Info("goimports completed",
			"file", filePath)
	}

	return hooks.HookOutput{Decision: "continue"}, nil
}