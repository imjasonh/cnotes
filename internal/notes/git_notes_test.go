package notes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// MockGitExecutor is a mock implementation of GitExecutor for testing
type MockGitExecutor struct {
	// Map of command patterns to responses
	responses map[string]mockResponse
	// Record of executed commands
	executed []executedCommand
}

type mockResponse struct {
	output []byte
	err    error
}

type executedCommand struct {
	dir  string
	args []string
}

func NewMockGitExecutor() *MockGitExecutor {
	return &MockGitExecutor{
		responses: make(map[string]mockResponse),
		executed:  []executedCommand{},
	}
}

func (m *MockGitExecutor) Execute(ctx context.Context, dir string, args ...string) ([]byte, error) {
	m.executed = append(m.executed, executedCommand{dir: dir, args: args})

	// Create a key from the command
	key := fmt.Sprintf("%v", args)

	if resp, ok := m.responses[key]; ok {
		return resp.output, resp.err
	}

	// Default response for unmatched commands
	return nil, fmt.Errorf("command not found: %v", args)
}

func (m *MockGitExecutor) SetResponse(args []string, output []byte, err error) {
	key := fmt.Sprintf("%v", args)
	m.responses[key] = mockResponse{output: output, err: err}
}

func (m *MockGitExecutor) GetExecutedCommands() []executedCommand {
	return m.executed
}

func TestNewNotesManager(t *testing.T) {
	workDir := "/test/dir"
	nm := NewNotesManager(workDir)

	if nm.workDir != workDir {
		t.Errorf("expected workDir %s, got %s", workDir, nm.workDir)
	}

	if nm.notesRef != "claude-conversations" {
		t.Errorf("expected notesRef claude-conversations, got %s", nm.notesRef)
	}

	if nm.git == nil {
		t.Error("expected git executor to be initialized")
	}
}

func TestNewNotesManagerWithExecutor(t *testing.T) {
	workDir := "/test/dir"
	mockGit := NewMockGitExecutor()
	nm := NewNotesManagerWithExecutor(workDir, mockGit)

	if nm.workDir != workDir {
		t.Errorf("expected workDir %s, got %s", workDir, nm.workDir)
	}

	if nm.git != mockGit {
		t.Error("expected custom git executor to be used")
	}
}

func TestSetNotesRef(t *testing.T) {
	nm := NewNotesManager("/test/dir")

	// Test setting a custom ref
	nm.SetNotesRef("custom-ref")
	if nm.notesRef != "custom-ref" {
		t.Errorf("expected notesRef custom-ref, got %s", nm.notesRef)
	}

	// Test empty ref (should not change)
	nm.SetNotesRef("")
	if nm.notesRef != "custom-ref" {
		t.Errorf("expected notesRef to remain custom-ref, got %s", nm.notesRef)
	}
}

func TestAddConversationNote(t *testing.T) {
	ctx := context.Background()
	mockGit := NewMockGitExecutor()
	nm := NewNotesManagerWithExecutor("/test/dir", mockGit)

	testNote := ConversationNote{
		SessionID:           "test-session-123",
		Timestamp:           time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		ConversationExcerpt: "User: Test this\nAssistant: Testing...",
		ToolsUsed:           []string{"Bash", "Read"},
		CommitContext:       "Fixed bug in feature X",
		ClaudeVersion:       "claude-3.5-sonnet-latest",
		LastEventTime:       time.Date(2023, 1, 1, 12, 30, 0, 0, time.UTC),
	}

	// Set up mock response for successful add
	expectedJSON, _ := json.MarshalIndent(testNote, "", "  ")
	mockGit.SetResponse(
		[]string{"notes", "--ref", "claude-conversations", "add", "-m", string(expectedJSON), "abc123"},
		[]byte{},
		nil,
	)

	// Add the note
	err := nm.AddConversationNote(ctx, "abc123", testNote)
	if err != nil {
		t.Fatalf("failed to add conversation note: %v", err)
	}

	// Verify the command was executed
	executed := mockGit.GetExecutedCommands()
	if len(executed) != 1 {
		t.Fatalf("expected 1 command, got %d", len(executed))
	}

	if executed[0].dir != "/test/dir" {
		t.Errorf("expected dir /test/dir, got %s", executed[0].dir)
	}

	// Should be: notes, --ref, claude-conversations, add, -m, <json>, abc123
	if len(executed[0].args) != 7 {
		t.Errorf("expected 7 args, got %d: %v", len(executed[0].args), executed[0].args)
	}
}

func TestAddConversationNoteError(t *testing.T) {
	ctx := context.Background()
	mockGit := NewMockGitExecutor()
	nm := NewNotesManagerWithExecutor("/test/dir", mockGit)

	testNote := ConversationNote{
		SessionID: "test-session",
		Timestamp: time.Now(),
	}

	// We don't set up a specific response, so it will use the default error
	// This simulates a git command failure

	// Try to add the note
	err := nm.AddConversationNote(ctx, "abc123", testNote)
	if err == nil {
		t.Error("expected error when adding duplicate note")
	}

	if !strings.Contains(err.Error(), "failed to add git note") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGetConversationNote(t *testing.T) {
	ctx := context.Background()
	mockGit := NewMockGitExecutor()
	nm := NewNotesManagerWithExecutor("/test/dir", mockGit)

	// Create expected note
	expectedNote := ConversationNote{
		SessionID:           "test-session-123",
		Timestamp:           time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		ConversationExcerpt: "User: Test this\nAssistant: Testing...",
		ToolsUsed:           []string{"Bash", "Read"},
		CommitContext:       "Fixed bug in feature X",
		ClaudeVersion:       "claude-3.5-sonnet-latest",
	}

	noteJSON, _ := json.Marshal(expectedNote)

	// Set up mock response
	mockGit.SetResponse(
		[]string{"notes", "--ref", "claude-conversations", "show", "abc123"},
		noteJSON,
		nil,
	)

	// Get the note
	note, err := nm.GetConversationNote(ctx, "abc123")
	if err != nil {
		t.Fatalf("failed to get conversation note: %v", err)
	}

	if note == nil {
		t.Fatal("expected note, got nil")
	}

	// Compare fields
	if note.SessionID != expectedNote.SessionID {
		t.Errorf("expected SessionID %s, got %s", expectedNote.SessionID, note.SessionID)
	}

	if note.ConversationExcerpt != expectedNote.ConversationExcerpt {
		t.Errorf("expected ConversationExcerpt %s, got %s", expectedNote.ConversationExcerpt, note.ConversationExcerpt)
	}
}

func TestGetConversationNoteNotFound(t *testing.T) {
	ctx := context.Background()
	mockGit := NewMockGitExecutor()
	nm := NewNotesManagerWithExecutor("/test/dir", mockGit)

	// Set up mock response for not found
	mockGit.SetResponse(
		[]string{"notes", "--ref", "claude-conversations", "show", "nonexistent"},
		nil,
		errors.New("no note found"),
	)

	// Try to get non-existent note
	note, err := nm.GetConversationNote(ctx, "nonexistent")
	if err != nil {
		t.Errorf("expected no error for non-existent note, got: %v", err)
	}

	if note != nil {
		t.Error("expected nil note for non-existent commit")
	}
}

func TestGetConversationNoteInvalidJSON(t *testing.T) {
	ctx := context.Background()
	mockGit := NewMockGitExecutor()
	nm := NewNotesManagerWithExecutor("/test/dir", mockGit)

	// Set up mock response with invalid JSON
	mockGit.SetResponse(
		[]string{"notes", "--ref", "claude-conversations", "show", "abc123"},
		[]byte("invalid json"),
		nil,
	)

	// Try to get note with invalid JSON
	note, err := nm.GetConversationNote(ctx, "abc123")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	if note != nil {
		t.Error("expected nil note for invalid JSON")
	}
}

func TestHasConversationNote(t *testing.T) {
	ctx := context.Background()
	mockGit := NewMockGitExecutor()
	nm := NewNotesManagerWithExecutor("/test/dir", mockGit)

	// Test with existing note
	noteJSON, _ := json.Marshal(ConversationNote{SessionID: "test"})
	mockGit.SetResponse(
		[]string{"notes", "--ref", "claude-conversations", "show", "abc123"},
		noteJSON,
		nil,
	)

	if !nm.HasConversationNote(ctx, "abc123") {
		t.Error("expected to have note")
	}

	// Test with non-existent note
	mockGit.SetResponse(
		[]string{"notes", "--ref", "claude-conversations", "show", "def456"},
		nil,
		errors.New("no note"),
	)

	if nm.HasConversationNote(ctx, "def456") {
		t.Error("expected not to have note")
	}
}

func TestExtractCommitHashFromOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "Standard format with branch",
			output:   "[main abc1234] Fixed bug in feature",
			expected: "abc1234",
		},
		{
			name:     "With leading/trailing whitespace",
			output:   "  [feature/test def5678] Added new feature  \n",
			expected: "def5678",
		},
		{
			name:     "Commit format",
			output:   "commit 1234567890abcdef",
			expected: "1234567890abcdef",
		},
		{
			name:     "Multiple lines with commit",
			output:   "Some output\n[main xyz789] Commit message\nMore output",
			expected: "xyz789",
		},
		{
			name:     "No commit hash",
			output:   "No commit information here",
			expected: "",
		},
		{
			name:     "Branch with slashes",
			output:   "[feature/new-thing abc123] Added feature",
			expected: "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractCommitHashFromOutput(tt.output)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIsGitCommitCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "Basic git commit",
			command:  "git commit",
			expected: true,
		},
		{
			name:     "Git commit with message",
			command:  "git commit -m \"Fix bug\"",
			expected: true,
		},
		{
			name:     "Git commit with add",
			command:  "git commit -am \"Add feature\"",
			expected: true,
		},
		{
			name:     "Git commit amend",
			command:  "git commit --amend",
			expected: true,
		},
		{
			name:     "Command with git commit in middle",
			command:  "cd /path && git commit -m \"test\"",
			expected: true,
		},
		{
			name:     "Git status (not commit)",
			command:  "git status",
			expected: false,
		},
		{
			name:     "Git add (not commit)",
			command:  "git add .",
			expected: false,
		},
		{
			name:     "Empty command",
			command:  "",
			expected: false,
		},
		{
			name:     "Command with whitespace",
			command:  "  git commit -m \"test\"  ",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGitCommitCommand(tt.command)
			if result != tt.expected {
				t.Errorf("expected %v for command %q, got %v", tt.expected, tt.command, result)
			}
		})
	}
}
