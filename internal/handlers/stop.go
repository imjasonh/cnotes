package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/imjasonh/hooks/internal/hooks"
)

func init() {
	hooks.RegisterHook(hooks.EventStop, "*", LogSessionCompletion)
}

// LogSessionCompletion is triggered when the main Claude Code agent finishes responding.
// This hook runs after Claude has completed its response but does not run if the session
// was interrupted by the user.
//
// Common use cases:
// - Session cleanup and logging
// - Performance metrics collection
// - Workflow completion notifications
// - Resource cleanup
// - Export session summaries
func LogSessionCompletion(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	slog.Info("Claude session completed",
		"session_id", input.SessionID,
		"cwd", input.CWD,
		"transcript_path", input.TranscriptPath,
		"timestamp", time.Now().Unix())

	// TODO: Future enhancements could include:
	// - Calculate session duration
	// - Count tools used in session
	// - Export conversation summary
	// - Send completion notifications
	// - Clean up temporary files
	// - Archive transcript

	return hooks.HookOutput{Decision: "approve"}, nil
}
