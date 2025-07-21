package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ConversationNote represents the structured data we store in git notes
type ConversationNote struct {
	SessionID           string    `json:"session_id"`
	Timestamp           time.Time `json:"timestamp"`
	ConversationExcerpt string    `json:"conversation_excerpt"`
	ToolsUsed           []string  `json:"tools_used"`
	CommitContext       string    `json:"commit_context"`
	ClaudeVersion       string    `json:"claude_version"`
}

// NotesManager handles git notes operations for Claude conversations
type NotesManager struct {
	notesRef string
	workDir  string
}

// NewNotesManager creates a new notes manager
func NewNotesManager(workDir string) *NotesManager {
	return &NotesManager{
		notesRef: "claude-conversations",
		workDir:  workDir,
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
	cmd := exec.CommandContext(ctx, "git", "notes", "--ref", nm.notesRef, "add", "-m", string(noteData), commitHash)
	cmd.Dir = nm.workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add git note: %w (output: %s)", err, string(output))
	}

	return nil
}

// GetConversationNote retrieves a conversation note for a specific commit
func (nm *NotesManager) GetConversationNote(ctx context.Context, commitHash string) (*ConversationNote, error) {
	cmd := exec.CommandContext(ctx, "git", "notes", "--ref", nm.notesRef, "show", commitHash)
	cmd.Dir = nm.workDir

	output, err := cmd.Output()
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

// IsGitCommitCommand checks if a bash command is a git commit
func IsGitCommitCommand(command string) bool {
	command = strings.TrimSpace(command)

	// Handle various git commit patterns
	patterns := []string{
		"git commit",
		"git commit -m",
		"git commit -am",
		"git commit --amend",
	}

	for _, pattern := range patterns {
		if strings.HasPrefix(command, pattern) {
			return true
		}
	}

	return false
}
