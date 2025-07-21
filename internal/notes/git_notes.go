package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GitExecutor defines the interface for executing git commands
type GitExecutor interface {
	Execute(ctx context.Context, dir string, args ...string) ([]byte, error)
}

// ConversationNote represents the structured data we store in git notes
type ConversationNote struct {
	SessionID           string    `json:"session_id"`
	Timestamp           time.Time `json:"timestamp"`
	ConversationExcerpt string    `json:"conversation_excerpt"`
	ToolsUsed           []string  `json:"tools_used"`
	CommitContext       string    `json:"commit_context"`
	ClaudeVersion       string    `json:"claude_version"`
	LastEventTime       time.Time `json:"last_event_time,omitempty"` // Track last processed event to avoid duplicates
}

// RealGitExecutor is the default implementation that runs actual git commands
type RealGitExecutor struct{}

// Execute runs a git command and returns its output
func (e *RealGitExecutor) Execute(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	return cmd.Output()
}

// NotesManager handles git notes operations for Claude conversations
type NotesManager struct {
	notesRef string
	workDir  string
	git      GitExecutor
}

// NewNotesManager creates a new notes manager
func NewNotesManager(workDir string) *NotesManager {
	return &NotesManager{
		notesRef: "claude-conversations",
		workDir:  workDir,
		git:      &RealGitExecutor{},
	}
}

// NewNotesManagerWithExecutor creates a new notes manager with a custom git executor
func NewNotesManagerWithExecutor(workDir string, git GitExecutor) *NotesManager {
	return &NotesManager{
		notesRef: "claude-conversations",
		workDir:  workDir,
		git:      git,
	}
}

// SetNotesRef updates the git notes reference name
func (nm *NotesManager) SetNotesRef(ref string) {
	if ref != "" {
		nm.notesRef = ref
	}
}

// AddConversationNote adds a conversation note to a specific commit
func (nm *NotesManager) AddConversationNote(ctx context.Context, commitHash string, note ConversationNote) error {
	// Marshal the note to JSON
	noteData, err := json.MarshalIndent(note, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal note: %w", err)
	}

	// Use git notes add command with custom ref
	_, err = nm.git.Execute(ctx, nm.workDir, "notes", "--ref", nm.notesRef, "add", "-m", string(noteData), commitHash)
	if err != nil {
		return fmt.Errorf("failed to add git note: %w", err)
	}

	return nil
}

// GetConversationNote retrieves a conversation note for a specific commit
func (nm *NotesManager) GetConversationNote(ctx context.Context, commitHash string) (*ConversationNote, error) {
	output, err := nm.git.Execute(ctx, nm.workDir, "notes", "--ref", nm.notesRef, "show", commitHash)
	if err != nil {
		// Note might not exist, which is normal
		return nil, nil
	}

	var note ConversationNote
	if err := json.Unmarshal(output, &note); err != nil {
		return nil, fmt.Errorf("failed to unmarshal note: %w", err)
	}

	return &note, nil
}

// HasConversationNote checks if a commit has a conversation note
func (nm *NotesManager) HasConversationNote(ctx context.Context, commitHash string) bool {
	note, _ := nm.GetConversationNote(ctx, commitHash)
	return note != nil
}

// ExtractCommitHashFromOutput attempts to extract commit hash from git command output
func ExtractCommitHashFromOutput(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for commit hash patterns
		if strings.HasPrefix(line, "[") && strings.Contains(line, "]") {
			// Format: [main abc1234] commit message
			parts := strings.Split(line, "]")
			if len(parts) > 0 {
				beforeBracket := strings.TrimSpace(strings.TrimPrefix(parts[0], "["))
				hashParts := strings.Split(beforeBracket, " ")
				if len(hashParts) > 1 {
					// Return the hash part (second element)
					return hashParts[1]
				}
			}
		}

		// Alternative format: commit abc1234567890abcdef
		if strings.HasPrefix(line, "commit ") {
			parts := strings.Split(line, " ")
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}

	return ""
}

// IsGitCommitCommand checks if a bash command contains a git commit
func IsGitCommitCommand(command string) bool {
	command = strings.TrimSpace(command)

	// Handle various git commit patterns - check if command contains any of these
	patterns := []string{
		"git commit",
		"git commit -m",
		"git commit -am",
		"git commit --amend",
	}

	for _, pattern := range patterns {
		if strings.Contains(command, pattern) {
			return true
		}
	}

	return false
}
