package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NotesBackup represents a backup of git notes
type NotesBackup struct {
	BackupTime time.Time                   `json:"backup_time"`
	NotesRef   string                      `json:"notes_ref"`
	Notes      map[string]ConversationNote `json:"notes"` // commit_hash -> note
}

// BackupAllNotes creates a backup of all notes in the specified ref
func (nm *NotesManager) BackupAllNotes(ctx context.Context) (*NotesBackup, error) {
	// Get list of all commits with notes
	output, err := nm.git.Execute(ctx, nm.workDir, "notes", "--ref", nm.notesRef, "list")
	if err != nil {
		// No notes exist, return empty backup
		return &NotesBackup{
			BackupTime: time.Now(),
			NotesRef:   nm.notesRef,
			Notes:      make(map[string]ConversationNote),
		}, nil
	}

	backup := &NotesBackup{
		BackupTime: time.Now(),
		NotesRef:   nm.notesRef,
		Notes:      make(map[string]ConversationNote),
	}

	// Parse the output to get note SHA and commit SHA pairs
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Format is: <note_sha> <commit_sha>
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}

		commitHash := parts[1]
		note, err := nm.GetConversationNote(ctx, commitHash)
		if err != nil || note == nil {
			continue
		}

		backup.Notes[commitHash] = *note
	}

	return backup, nil
}

// SaveBackupToFile saves a notes backup to a JSON file
func (nm *NotesManager) SaveBackupToFile(backup *NotesBackup, filename string) error {
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup: %w", err)
	}

	if !filepath.IsAbs(filename) {
		filename = filepath.Join(nm.workDir, filename)
	}

	return os.WriteFile(filename, data, 0644)
}

// LoadBackupFromFile loads a notes backup from a JSON file
func (nm *NotesManager) LoadBackupFromFile(filename string) (*NotesBackup, error) {
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(nm.workDir, filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	var backup NotesBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	return &backup, nil
}

// RestoreNotesFromBackup restores notes from a backup, trying to match them to current commits
func (nm *NotesManager) RestoreNotesFromBackup(ctx context.Context, backup *NotesBackup) error {
	restored := 0
	skipped := 0

	for commitHash, note := range backup.Notes {
		// Check if the commit still exists
		_, err := nm.git.Execute(ctx, nm.workDir, "cat-file", "-e", commitHash)
		if err != nil {
			// Commit doesn't exist anymore, skip
			skipped++
			continue
		}

		// Check if note already exists
		if nm.HasConversationNote(ctx, commitHash) {
			// Note already exists, skip
			skipped++
			continue
		}

		// Restore the note
		if err := nm.AddConversationNote(ctx, commitHash, note); err != nil {
			return fmt.Errorf("failed to restore note for commit %s: %w", commitHash, err)
		}

		restored++
	}

	fmt.Printf("Notes restoration complete: %d restored, %d skipped\n", restored, skipped)
	return nil
}

// CreateRebaseBackup creates a timestamped backup before potentially destructive operations
func (nm *NotesManager) CreateRebaseBackup(ctx context.Context) (string, error) {
	backup, err := nm.BackupAllNotes(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf(".claude-notes-backup-%s.json", timestamp)

	if err := nm.SaveBackupToFile(backup, filename); err != nil {
		return "", fmt.Errorf("failed to save backup: %w", err)
	}

	return filename, nil
}
