package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/imjasonh/cnotes/internal/config"
	conv "github.com/imjasonh/cnotes/internal/context"
	"github.com/imjasonh/cnotes/internal/notes"
	"github.com/spf13/cobra"
)

var (
	debug   bool
	rootCmd = &cobra.Command{
		Use:   "cnotes",
		Short: "Git notes for Claude conversations",
		Long: `cnotes automatically captures Claude conversation context in git notes.
		
When called by Claude Code hooks, it detects git commit commands and attaches
conversation context as git notes for easy reference later.`,
		RunE: runHook,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug {
				opts := &slog.HandlerOptions{Level: slog.LevelDebug}
				handler := slog.NewTextHandler(os.Stderr, opts)
				slog.SetDefault(slog.New(handler))
			}
		},
	}
)

// HookInput represents the input from Claude Code hooks
type HookInput struct {
	SessionID      string          `json:"session_id"`
	TranscriptPath string          `json:"transcript_path"`
	CWD            string          `json:"cwd"`
	HookEventName  string          `json:"hook_event_name"`
	ToolName       string          `json:"tool_name,omitempty"`
	ToolInput      json.RawMessage `json:"tool_input,omitempty"`
	ToolResponse   json.RawMessage `json:"tool_response,omitempty"`
}

// BashToolInput represents bash tool parameters
type BashToolInput struct {
	Command     string `json:"command"`
	Description string `json:"description,omitempty"`
}

// HookOutput represents the response to Claude Code
type HookOutput struct {
	Decision string `json:"decision,omitempty"`
}

func runHook(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Check if stdin has data available with a short timeout
	inputBytes, err := readStdinWithTimeout(2 * time.Second)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	if len(inputBytes) == 0 {
		return fmt.Errorf("no input received - cnotes expects JSON input from Claude Code hooks")
	}

	// Parse hook input
	var input HookInput
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		return fmt.Errorf("failed to parse input: %w", err)
	}

	// Only handle PostToolUse events for Bash commands
	if input.HookEventName != "PostToolUse" || input.ToolName != "Bash" {
		// For all other events, just approve
		return writeOutput(HookOutput{Decision: "approve"})
	}

	// Extract bash command
	var bashInput BashToolInput
	if err := json.Unmarshal(input.ToolInput, &bashInput); err != nil {
		return writeOutput(HookOutput{Decision: "approve"})
	}

	// Check if this is a git commit command
	if !isGitCommitCommand(bashInput.Command) {
		return writeOutput(HookOutput{Decision: "approve"})
	}

	// Load configuration
	cfg := config.LoadNotesConfig(input.CWD)
	if !cfg.Enabled {
		return writeOutput(HookOutput{Decision: "approve"})
	}

	// Process the git commit and attach notes
	if err := processGitCommit(ctx, input, bashInput); err != nil {
		slog.Error("failed to process git commit", "error", err)
		// Don't fail the hook, just log the error
	}

	return writeOutput(HookOutput{Decision: "approve"})
}

func processGitCommit(ctx context.Context, input HookInput, bashInput BashToolInput) error {
	// Extract git output from tool response
	var gitOutput string
	if len(input.ToolResponse) > 0 {
		var toolResponse struct {
			Stdout string `json:"stdout"`
		}
		if err := json.Unmarshal(input.ToolResponse, &toolResponse); err == nil {
			gitOutput = toolResponse.Stdout
		}
	}

	if gitOutput == "" {
		return fmt.Errorf("no git output found")
	}

	// Extract commit hash
	commitHash := extractCommitHash(gitOutput)
	if commitHash == "" {
		return fmt.Errorf("could not extract commit hash")
	}

	// Create notes manager and load config
	notesManager := notes.NewNotesManager(input.CWD)
	cfg := config.LoadNotesConfig(input.CWD)
	notesManager.SetNotesRef(cfg.NotesRef)

	// Check if note already exists
	if notesManager.HasConversationNote(ctx, commitHash) {
		return nil
	}

	// Get the timestamp of the previous commit in this session
	previousCommitTime := getLastCommitTimeForSession(ctx, notesManager, input.SessionID)
	
	// Small delay to ensure transcript is written
	time.Sleep(100 * time.Millisecond)
	
	// Extract conversation context since the last commit
	contextExtractor := conv.NewContextExtractor(cfg)
	conversationContext, err := contextExtractor.ExtractContextSince(input.TranscriptPath, input.SessionID, previousCommitTime)
	if err != nil {
		return fmt.Errorf("failed to extract conversation context: %w", err)
	}

	// Create conversation excerpt
	excerpt := contextExtractor.CreateExcerpt(conversationContext)

	// Collect tools used
	toolsUsed := []string{"Bash"}
	for _, interaction := range conversationContext.ToolInteractions {
		if interaction.Tool != "" && !contains(toolsUsed, interaction.Tool) {
			toolsUsed = append(toolsUsed, interaction.Tool)
		}
	}

	// Create conversation note
	note := notes.ConversationNote{
		SessionID:           input.SessionID,
		Timestamp:           time.Now(),
		ConversationExcerpt: excerpt,
		ToolsUsed:           toolsUsed,
		CommitContext:       buildCommitContext(bashInput.Command, gitOutput),
		ClaudeVersion:       "claude-sonnet-4-20250514",
	}

	// Add the note
	if err := notesManager.AddConversationNote(ctx, commitHash, note); err != nil {
		return fmt.Errorf("failed to add conversation note: %w", err)
	}

	slog.Info("attached conversation context to commit",
		"commit", commitHash,
		"session_id", input.SessionID)

	return nil
}

func isGitCommitCommand(command string) bool {
	command = strings.TrimSpace(command)
	patterns := []string{"git commit"}
	for _, pattern := range patterns {
		if strings.Contains(command, pattern) {
			return true
		}
	}
	return false
}

func extractCommitHash(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.Contains(line, "]") {
			parts := strings.Split(line, "]")
			if len(parts) > 0 {
				beforeBracket := strings.TrimSpace(strings.TrimPrefix(parts[0], "["))
				hashParts := strings.Split(beforeBracket, " ")
				if len(hashParts) > 1 {
					return hashParts[1]
				}
			}
		}
	}
	return ""
}

func buildCommitContext(command, output string) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Git command: %s", command))

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "[") && strings.Contains(line, "]") {
			parts = append(parts, fmt.Sprintf("Result: %s", line))
			break
		}
	}

	return strings.Join(parts, "\n")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// getLastCommitTimeForSession finds the most recent commit time for this session
func getLastCommitTimeForSession(ctx context.Context, notesManager *notes.NotesManager, sessionID string) time.Time {
	// For now, let's use a simpler approach - get the time of the previous commit
	// This works well when commits are made sequentially in a session
	cmd := exec.Command("git", "log", "-1", "--format=%cI", "HEAD~1")
	output, err := cmd.Output()
	if err != nil {
		// No previous commit or error, return zero time
		return time.Time{}
	}
	
	timeStr := strings.TrimSpace(string(output))
	commitTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}
	}
	
	// Add a larger buffer to ensure we capture user prompts that triggered the work
	// User prompts often happen 30-60 seconds before the commit
	return commitTime.Add(-60 * time.Second)
}

// readStdinWithTimeout reads from stdin with a timeout
func readStdinWithTimeout(timeout time.Duration) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}

	ch := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(os.Stdin)
		ch <- result{data, err}
	}()

	select {
	case res := <-ch:
		return res.data, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for input after %v", timeout)
	}
}

func writeOutput(output HookOutput) error {
	outputBytes, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	fmt.Println(string(outputBytes))
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
}
