package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/imjasonh/hooks/internal/hooks"
)

func init() {
	hooks.RegisterHook(hooks.EventPreCompact, "*", HandlePreCompact)
}

// HandlePreCompact is triggered before Claude Code runs a compact operation to
// reduce context window size. This can happen manually via /compact command
// or automatically when the context window fills up.
//
// The matcher indicates the compaction trigger:
// - "manual": User ran /compact command
// - "auto": Automatic compaction due to context window limit
//
// Common use cases:
// - Save conversation state before compaction
// - Warn user about potential data loss
// - Export important context to external files
// - Create conversation backups
// - Log compaction events for analysis
func HandlePreCompact(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	// Determine compaction type from the tool name/matcher
	compactionType := "unknown"
	if input.Tool == "manual" || input.ToolName == "manual" {
		compactionType = "manual"
	} else if input.Tool == "auto" || input.ToolName == "auto" {
		compactionType = "auto"
	}

	slog.Info("pre-compaction hook triggered",
		"session_id", input.SessionID,
		"compaction_type", compactionType,
		"cwd", input.CWD,
		"transcript_path", input.TranscriptPath,
		"timestamp", time.Now().Unix())

	// Add context warning for the user
	var warningContext string
	if compactionType == "manual" {
		warningContext = "Manual compaction requested - some conversation history will be summarized"
	} else if compactionType == "auto" {
		warningContext = "Automatic compaction triggered - context window limit reached"
	} else {
		warningContext = "Context compaction about to occur"
	}

	// TODO: Future enhancements could include:
	// - Export full conversation before compaction
	// - Save important code snippets or commands
	// - Create conversation timeline backup
	// - Allow user to specify what to preserve
	// - Integration with external note-taking systems
	// - Automatic git commit of current work
	// - Context importance scoring and preservation

	return hooks.HookOutput{
		Decision:          "approve",
		AdditionalContext: fmt.Sprintf("🔄 %s", warningContext),
	}, nil
}
