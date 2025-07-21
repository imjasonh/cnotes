package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/imjasonh/hooks/internal/hooks"
)

func LogToolUsage(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	switch input.Tool {
	case "Bash":
		if bashInput, err := input.GetBashInput(); err == nil {
			slog.Info("bash command executed",
				"command", bashInput.Command,
				"session_id", input.SessionID,
				"cwd", input.CWD,
				"timestamp", time.Now().Unix())
		}
	case "Write", "Edit", "MultiEdit", "Read":
		if fileInput, err := input.GetFileInput(); err == nil {
			if input.Tool == "Read" {
				slog.Info("file read",
					"file", fileInput.FilePath,
					"session_id", input.SessionID,
					"timestamp", time.Now().Unix())
			} else {
				slog.Info("file modified",
					"tool", input.Tool,
					"file", fileInput.FilePath,
					"session_id", input.SessionID,
					"timestamp", time.Now().Unix())
			}
		}
	default:
		slog.Info("tool used",
			"tool", input.Tool,
			"session_id", input.SessionID,
			"timestamp", time.Now().Unix())
	}

	return hooks.HookOutput{Decision: "approve"}, nil
}
