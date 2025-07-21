package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/imjasonh/hooks/internal/hooks"
)

var dangerousCommands = []string{
	"rm -rf /",
	"rm -rf /*",
	":(){ :|:& };:",
	"mkfs.",
	"dd if=/dev/zero",
	"> /dev/sda",
	"wget http",
	"curl http",
}

func ValidateBashCommand(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	cmd, ok := input.ToolUseRequest.Parameters["command"].(string)
	if !ok {
		return hooks.HookOutput{Decision: "continue"}, nil
	}

	for _, dangerous := range dangerousCommands {
		if strings.Contains(cmd, dangerous) {
			slog.Warn("blocked dangerous command",
				"command", cmd,
				"pattern", dangerous)
			return hooks.HookOutput{
				Decision: "block",
				Reason:   fmt.Sprintf("Command contains dangerous pattern: %s", dangerous),
			}, nil
		}
	}

	if strings.Contains(cmd, "sudo") && !strings.Contains(cmd, "sudo -n") {
		slog.Info("modifying sudo command to non-interactive")
		return hooks.HookOutput{
			Decision: "continue",
			ModifiedParameters: map[string]interface{}{
				"command": strings.ReplaceAll(cmd, "sudo", "sudo -n"),
			},
		}, nil
	}

	return hooks.HookOutput{Decision: "continue"}, nil
}

var sensitiveFiles = []string{
	".env",
	".aws/credentials",
	".ssh/id_rsa",
	"secrets",
	"password",
	"token",
	"key.pem",
}

func PreventSensitiveFileEdits(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	path, ok := input.ToolUseRequest.Parameters["file_path"].(string)
	if !ok {
		return hooks.HookOutput{Decision: "continue"}, nil
	}

	lowerPath := strings.ToLower(path)
	for _, sensitive := range sensitiveFiles {
		if strings.Contains(lowerPath, strings.ToLower(sensitive)) {
			slog.Warn("blocked sensitive file edit",
				"file", path,
				"pattern", sensitive)
			return hooks.HookOutput{
				Decision: "block",
				Reason:   fmt.Sprintf("Cannot edit sensitive file: %s", path),
			}, nil
		}
	}

	return hooks.HookOutput{Decision: "continue"}, nil
}