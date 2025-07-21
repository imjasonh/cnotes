package notes

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackupAllNotes(t *testing.T) {
	ctx := context.Background()
	mockGit := NewMockGitExecutor()
	nm := NewNotesManagerWithExecutor("/test/dir", mockGit)

	t.Run("with existing notes", func(t *testing.T) {
		// Mock git notes list output
		listOutput := "note-sha1 commit1\nnote-sha2 commit2\n"
		mockGit.SetResponse(
			[]string{"notes", "--ref", "claude-conversations", "list"},
			[]byte(listOutput),
			nil,
		)

		// Mock getting note for commit1
		note1 := ConversationNote{
			SessionID:           "session1",
			Timestamp:           time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			ConversationExcerpt: "First conversation",
		}
		note1JSON, _ := json.Marshal(note1)
		mockGit.SetResponse(
			[]string{"notes", "--ref", "claude-conversations", "show", "commit1"},
			note1JSON,
			nil,
		)

		// Mock getting note for commit2
		note2 := ConversationNote{
			SessionID:           "session2",
			Timestamp:           time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
			ConversationExcerpt: "Second conversation",
		}
		note2JSON, _ := json.Marshal(note2)
		mockGit.SetResponse(
			[]string{"notes", "--ref", "claude-conversations", "show", "commit2"},
			note2JSON,
			nil,
		)

		// Perform backup
		backup, err := nm.BackupAllNotes(ctx)
		if err != nil {
			t.Fatalf("failed to backup notes: %v", err)
		}

		// Verify backup
		if backup.NotesRef != "claude-conversations" {
			t.Errorf("expected notes ref claude-conversations, got %s", backup.NotesRef)
		}

		if len(backup.Notes) != 2 {
			t.Errorf("expected 2 notes, got %d", len(backup.Notes))
		}

		if backup.Notes["commit1"].SessionID != "session1" {
			t.Errorf("expected session1 for commit1")
		}

		if backup.Notes["commit2"].SessionID != "session2" {
			t.Errorf("expected session2 for commit2")
		}
	})

	t.Run("with no notes", func(t *testing.T) {
		mockGit := NewMockGitExecutor()
		nm := NewNotesManagerWithExecutor("/test/dir", mockGit)

		// Mock empty notes list
		mockGit.SetResponse(
			[]string{"notes", "--ref", "claude-conversations", "list"},
			nil,
			errors.New("no notes found"),
		)

		backup, err := nm.BackupAllNotes(ctx)
		if err != nil {
			t.Fatalf("failed to backup notes: %v", err)
		}

		if len(backup.Notes) != 0 {
			t.Errorf("expected 0 notes, got %d", len(backup.Notes))
		}
	})

	t.Run("with malformed list output", func(t *testing.T) {
		mockGit := NewMockGitExecutor()
		nm := NewNotesManagerWithExecutor("/test/dir", mockGit)

		// Mock malformed git notes list output
		listOutput := "invalid-format\nnote-sha commit1\n"
		mockGit.SetResponse(
			[]string{"notes", "--ref", "claude-conversations", "list"},
			[]byte(listOutput),
			nil,
		)

		// Mock getting note for commit1
		note1 := ConversationNote{SessionID: "session1"}
		note1JSON, _ := json.Marshal(note1)
		mockGit.SetResponse(
			[]string{"notes", "--ref", "claude-conversations", "show", "commit1"},
			note1JSON,
			nil,
		)

		backup, err := nm.BackupAllNotes(ctx)
		if err != nil {
			t.Fatalf("failed to backup notes: %v", err)
		}

		// Should only have 1 note (skipped malformed line)
		if len(backup.Notes) != 1 {
			t.Errorf("expected 1 note, got %d", len(backup.Notes))
		}
	})
}

func TestSaveAndLoadBackupFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "cnotes-backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	nm := NewNotesManager(tempDir)

	// Create a test backup
	backup := &NotesBackup{
		BackupTime: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		NotesRef:   "claude-conversations",
		Notes: map[string]ConversationNote{
			"commit1": {
				SessionID:           "session1",
				Timestamp:           time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
				ConversationExcerpt: "Test conversation",
			},
		},
	}

	t.Run("save and load with relative path", func(t *testing.T) {
		filename := "test-backup.json"

		// Save backup
		if err := nm.SaveBackupToFile(backup, filename); err != nil {
			t.Fatalf("failed to save backup: %v", err)
		}

		// Verify file exists
		fullPath := filepath.Join(tempDir, filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Error("backup file was not created")
		}

		// Load backup
		loaded, err := nm.LoadBackupFromFile(filename)
		if err != nil {
			t.Fatalf("failed to load backup: %v", err)
		}

		// Verify loaded data
		if loaded.NotesRef != backup.NotesRef {
			t.Errorf("expected notes ref %s, got %s", backup.NotesRef, loaded.NotesRef)
		}

		if len(loaded.Notes) != len(backup.Notes) {
			t.Errorf("expected %d notes, got %d", len(backup.Notes), len(loaded.Notes))
		}

		if loaded.Notes["commit1"].SessionID != "session1" {
			t.Error("loaded note data doesn't match")
		}
	})

	t.Run("save and load with absolute path", func(t *testing.T) {
		filename := filepath.Join(tempDir, "absolute-backup.json")

		// Save backup
		if err := nm.SaveBackupToFile(backup, filename); err != nil {
			t.Fatalf("failed to save backup: %v", err)
		}

		// Load backup
		loaded, err := nm.LoadBackupFromFile(filename)
		if err != nil {
			t.Fatalf("failed to load backup: %v", err)
		}

		// Verify loaded data
		if len(loaded.Notes) != 1 {
			t.Errorf("expected 1 note, got %d", len(loaded.Notes))
		}
	})

	t.Run("load non-existent file", func(t *testing.T) {
		_, err := nm.LoadBackupFromFile("non-existent.json")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("load invalid JSON", func(t *testing.T) {
		// Create invalid JSON file
		invalidFile := filepath.Join(tempDir, "invalid.json")
		if err := os.WriteFile(invalidFile, []byte("invalid json"), 0644); err != nil {
			t.Fatalf("failed to write invalid file: %v", err)
		}

		_, err := nm.LoadBackupFromFile("invalid.json")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestRestoreNotesFromBackup(t *testing.T) {
	ctx := context.Background()

	t.Run("restore with existing commits", func(t *testing.T) {
		mockGit := NewMockGitExecutor()
		nm := NewNotesManagerWithExecutor("/test/dir", mockGit)

		backup := &NotesBackup{
			BackupTime: time.Now(),
			NotesRef:   "claude-conversations",
			Notes: map[string]ConversationNote{
				"commit1": {SessionID: "session1"},
				"commit2": {SessionID: "session2"},
			},
		}

		// Mock commit existence checks
		mockGit.SetResponse(
			[]string{"cat-file", "-e", "commit1"},
			[]byte{},
			nil,
		)
		mockGit.SetResponse(
			[]string{"cat-file", "-e", "commit2"},
			[]byte{},
			nil,
		)

		// Mock checking for existing notes (none exist)
		mockGit.SetResponse(
			[]string{"notes", "--ref", "claude-conversations", "show", "commit1"},
			nil,
			errors.New("no note"),
		)
		mockGit.SetResponse(
			[]string{"notes", "--ref", "claude-conversations", "show", "commit2"},
			nil,
			errors.New("no note"),
		)

		// Mock successful note additions
		note1JSON, _ := json.MarshalIndent(backup.Notes["commit1"], "", "  ")
		mockGit.SetResponse(
			[]string{"notes", "--ref", "claude-conversations", "add", "-m", string(note1JSON), "commit1"},
			[]byte{},
			nil,
		)
		note2JSON, _ := json.MarshalIndent(backup.Notes["commit2"], "", "  ")
		mockGit.SetResponse(
			[]string{"notes", "--ref", "claude-conversations", "add", "-m", string(note2JSON), "commit2"},
			[]byte{},
			nil,
		)

		// Restore
		err := nm.RestoreNotesFromBackup(ctx, backup)
		if err != nil {
			t.Fatalf("failed to restore notes: %v", err)
		}

		// Verify all commands were executed
		executed := mockGit.GetExecutedCommands()
		if len(executed) < 6 { // 2 cat-file, 2 show, 2 add
			t.Errorf("expected at least 6 commands, got %d", len(executed))
		}
	})

	t.Run("skip non-existent commits", func(t *testing.T) {
		mockGit := NewMockGitExecutor()
		nm := NewNotesManagerWithExecutor("/test/dir", mockGit)

		backup := &NotesBackup{
			Notes: map[string]ConversationNote{
				"missing": {SessionID: "session1"},
				"exists":  {SessionID: "session2"},
			},
		}

		// Mock commit existence checks
		mockGit.SetResponse(
			[]string{"cat-file", "-e", "missing"},
			nil,
			errors.New("not found"),
		)
		mockGit.SetResponse(
			[]string{"cat-file", "-e", "exists"},
			[]byte{},
			nil,
		)

		// Mock checking for existing note
		mockGit.SetResponse(
			[]string{"notes", "--ref", "claude-conversations", "show", "exists"},
			nil,
			errors.New("no note"),
		)

		// Mock successful note addition
		noteJSON, _ := json.MarshalIndent(backup.Notes["exists"], "", "  ")
		mockGit.SetResponse(
			[]string{"notes", "--ref", "claude-conversations", "add", "-m", string(noteJSON), "exists"},
			[]byte{},
			nil,
		)

		// Restore
		err := nm.RestoreNotesFromBackup(ctx, backup)
		if err != nil {
			t.Fatalf("failed to restore notes: %v", err)
		}
	})

	t.Run("skip existing notes", func(t *testing.T) {
		mockGit := NewMockGitExecutor()
		nm := NewNotesManagerWithExecutor("/test/dir", mockGit)

		backup := &NotesBackup{
			Notes: map[string]ConversationNote{
				"commit1": {SessionID: "session1"},
			},
		}

		// Mock commit exists
		mockGit.SetResponse(
			[]string{"cat-file", "-e", "commit1"},
			[]byte{},
			nil,
		)

		// Mock note already exists
		existingNote := ConversationNote{SessionID: "existing"}
		existingJSON, _ := json.Marshal(existingNote)
		mockGit.SetResponse(
			[]string{"notes", "--ref", "claude-conversations", "show", "commit1"},
			existingJSON,
			nil,
		)

		// Restore (should skip since note exists)
		err := nm.RestoreNotesFromBackup(ctx, backup)
		if err != nil {
			t.Fatalf("failed to restore notes: %v", err)
		}

		// Verify no add command was executed
		executed := mockGit.GetExecutedCommands()
		for _, cmd := range executed {
			if len(cmd.args) > 0 && cmd.args[0] == "add" {
				t.Error("should not have attempted to add note that already exists")
			}
		}
	})
}

func TestCreateRebaseBackup(t *testing.T) {
	ctx := context.Background()

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "cnotes-rebase-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockGit := NewMockGitExecutor()
	nm := NewNotesManagerWithExecutor(tempDir, mockGit)

	// Mock notes list
	mockGit.SetResponse(
		[]string{"notes", "--ref", "claude-conversations", "list"},
		[]byte("note-sha commit1\n"),
		nil,
	)

	// Mock getting note
	note := ConversationNote{SessionID: "session1"}
	noteJSON, _ := json.Marshal(note)
	mockGit.SetResponse(
		[]string{"notes", "--ref", "claude-conversations", "show", "commit1"},
		noteJSON,
		nil,
	)

	// Create rebase backup
	filename, err := nm.CreateRebaseBackup(ctx)
	if err != nil {
		t.Fatalf("failed to create rebase backup: %v", err)
	}

	// Verify filename format
	if !strings.HasPrefix(filename, ".claude-notes-backup-") {
		t.Errorf("unexpected filename format: %s", filename)
	}

	if !strings.HasSuffix(filename, ".json") {
		t.Errorf("expected .json extension: %s", filename)
	}

	// Verify file exists
	fullPath := filepath.Join(tempDir, filename)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("backup file was not created")
	}

	// Verify file contents
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read backup file: %v", err)
	}

	var backup NotesBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		t.Fatalf("failed to parse backup: %v", err)
	}

	if len(backup.Notes) != 1 {
		t.Errorf("expected 1 note in backup, got %d", len(backup.Notes))
	}
}
