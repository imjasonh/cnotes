package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/imjasonh/hooks/internal/hooks"
)

func LogToolUsage(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	params := input.ToolUseRequest.Parameters
	
	switch input.Tool {
	case "Bash":
		if cmd, ok := params["command"].(string); ok {
			slog.Info("bash command executed",
				"command", cmd,
				"session_id", input.SessionID,
				"cwd", input.CWD,
				"timestamp", time.Now().Unix())
		}
	case "Write", "Edit", "MultiEdit":
		if path, ok := params["file_path"].(string); ok {
			slog.Info("file modified",
				"tool", input.Tool,
				"file", path,
				"session_id", input.SessionID,
				"timestamp", time.Now().Unix())
		}
	case "Read":
		if path, ok := params["file_path"].(string); ok {
			slog.Info("file read",
				"file", path,
				"session_id", input.SessionID,
				"timestamp", time.Now().Unix())
		}
	default:
		slog.Info("tool used",
			"tool", input.Tool,
			"session_id", input.SessionID,
			"timestamp", time.Now().Unix())
	}

	return hooks.HookOutput{Decision: "approve"}, nil
}