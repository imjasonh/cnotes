package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/imjasonh/hooks/internal/hooks"
)

func init() {
	hooks.RegisterHook(hooks.EventSubagentStop, "*", LogSubagentCompletion)
}

// LogSubagentCompletion is triggered when a Claude Code subagent (Task tool call)
// finishes responding. Subagents are created when Claude uses the Task tool to
// delegate work to another Claude instance.
//
// Common use cases:
// - Track nested workflow completion
// - Monitor subagent performance
// - Resource cleanup after Task operations
// - Debug complex multi-agent workflows
// - Collect metrics on task delegation
func LogSubagentCompletion(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	slog.Info("Claude subagent completed",
		"session_id", input.SessionID,
		"cwd", input.CWD,
		"transcript_path", input.TranscriptPath,
		"timestamp", time.Now().Unix())

	// TODO: Future enhancements could include:
	// - Track subagent hierarchy depth
	// - Measure subagent execution time
	// - Collect subagent success/failure rates
	// - Clean up subagent-specific resources
	// - Aggregate results from multiple subagents
	// - Send notifications for long-running subagents
	// - Export subagent conversation logs

	return hooks.HookOutput{Decision: "approve"}, nil
}
