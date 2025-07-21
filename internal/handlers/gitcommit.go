package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/imjasonh/hooks/internal/config"
	conv "github.com/imjasonh/hooks/internal/context"
	"github.com/imjasonh/hooks/internal/hooks"
	"github.com/imjasonh/hooks/internal/notes"
)

func init() {
	hooks.RegisterHook(hooks.EventPostToolUse, "Bash", AttachConversationToCommit)
}

// AttachConversationToCommit detects git commit commands and attaches conversation context
// as git notes. This runs as a PostToolUse hook so we can capture the commit hash from
// the git command output.
func AttachConversationToCommit(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	// Get the bash command details
	bashInput, err := input.GetBashInput()
	if err != nil {
		// Not a bash command, skip
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	// Check if this is a git commit command
	if !notes.IsGitCommitCommand(bashInput.Command) {
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	// Load configuration to check if notes are enabled
	cfg := config.LoadNotesConfig(input.CWD)
	if !cfg.Enabled {
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	// Extract commit hash from tool response/result
	var gitOutput string
	if len(input.ToolResponse) > 0 {
		// Parse the JSON response to extract stdout
		var toolResponse struct {
			Stdout      string `json:"stdout"`
			Stderr      string `json:"stderr"`
			Interrupted bool   `json:"interrupted"`
			IsImage     bool   `json:"isImage"`
		}
		if err := json.Unmarshal(input.ToolResponse, &toolResponse); err == nil {
			gitOutput = toolResponse.Stdout
		} else {
			// Fallback to raw string if JSON parsing fails
			gitOutput = string(input.ToolResponse)
		}
	} else if len(input.ToolUseResult) > 0 {
		gitOutput = string(input.ToolUseResult)
	}

	if gitOutput == "" {
		slog.Debug("no git output found for commit command", "command", bashInput.Command)
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	// Extract commit hash from git output
	commitHash := notes.ExtractCommitHashFromOutput(gitOutput)
	if commitHash == "" {
		slog.Debug("could not extract commit hash from git output", "output", gitOutput)
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	// Create notes manager with configuration
	notesManager := notes.NewNotesManager(input.CWD)
	notesManager.SetNotesRef(cfg.NotesRef)

	// Check if we already have a note for this commit
	if notesManager.HasConversationNote(ctx, commitHash) {
		slog.Debug("commit already has conversation note", "commit", commitHash)
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	// Extract conversation context
	contextExtractor := conv.NewContextExtractor()
	conversationContext, err := contextExtractor.ExtractRecentContext(input.TranscriptPath, input.SessionID)
	if err != nil {
		slog.Error("failed to extract conversation context", "error", err)
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	// Create conversation excerpt
	excerpt := contextExtractor.CreateExcerpt(conversationContext)

	// Collect tools used in this session (from conversation context)
	toolsUsed := make([]string, 0)
	toolsUsed = append(toolsUsed, "Bash") // Current tool
	for _, interaction := range conversationContext.ToolInteractions {
		if interaction.Tool != "" {
			// Add unique tools
			found := false
			for _, existing := range toolsUsed {
				if existing == interaction.Tool {
					found = true
					break
				}
			}
			if !found {
				toolsUsed = append(toolsUsed, interaction.Tool)
			}
		}
	}

	// Create conversation note
	note := notes.ConversationNote{
		SessionID:           input.SessionID,
		Timestamp:           time.Now(),
		ConversationExcerpt: excerpt,
		ToolsUsed:           toolsUsed,
		CommitContext:       buildCommitContext(bashInput.Command, gitOutput),
		ClaudeVersion:       "claude-sonnet-4-20250514", // Could be made configurable
	}

	// Add the note to git
	if err := notesManager.AddConversationNote(ctx, commitHash, note); err != nil {
		slog.Error("failed to add conversation note to commit",
			"error", err,
			"commit", commitHash,
			"session_id", input.SessionID)
		// Don't fail the hook, just log the error
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	slog.Info("attached conversation context to commit",
		"commit", commitHash,
		"session_id", input.SessionID,
		"tools_used", strings.Join(toolsUsed, ", "),
		"excerpt_length", len(excerpt))

	return hooks.HookOutput{Decision: "approve"}, nil
}

// buildCommitContext creates a summary of the commit context
func buildCommitContext(command, output string) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Git command: %s", command))

	// Extract key information from git output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Include commit summary line
		if strings.Contains(line, "[") && strings.Contains(line, "]") {
			parts = append(parts, fmt.Sprintf("Result: %s", line))
			break
		}

		// Include first meaningful line
		if !strings.HasPrefix(line, "On branch") && !strings.HasPrefix(line, "Your branch") {
			parts = append(parts, fmt.Sprintf("Output: %s", line))
			if len(parts) >= 3 { // Limit context size
				break
			}
		}
	}

	return strings.Join(parts, "\n")
}
