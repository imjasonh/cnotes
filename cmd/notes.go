package cmd

import (
	"context"
	"fmt"

	"github.com/imjasonh/hooks/internal/notes"
	"github.com/spf13/cobra"
)

var notesCmd = &cobra.Command{
	Use:   "notes",
	Short: "Manage git notes backups and restoration",
	Long:  `Commands for backing up, restoring, and managing Claude conversation notes.`,
}

var backupCmd = &cobra.Command{
	Use:   "backup [filename]",
	Short: "Backup all conversation notes to a JSON file",
	Long: `Creates a backup of all conversation notes attached to commits.
If no filename is provided, creates a timestamped backup file.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		notesManager := notes.NewNotesManager(".")

		var filename string
		if len(args) > 0 {
			filename = args[0]
		} else {
			var err error
			filename, err = notesManager.CreateRebaseBackup(ctx)
			if err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}
		}

		backup, err := notesManager.BackupAllNotes(ctx)
		if err != nil {
			return fmt.Errorf("failed to backup notes: %w", err)
		}

		if err := notesManager.SaveBackupToFile(backup, filename); err != nil {
			return fmt.Errorf("failed to save backup file: %w", err)
		}

		fmt.Printf("✅ Backed up %d conversation notes to %s\n", len(backup.Notes), filename)
		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore <filename>",
	Short: "Restore conversation notes from a backup file",
	Long: `Restores conversation notes from a previously created backup file.
Only restores notes for commits that still exist and don't already have notes.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		notesManager := notes.NewNotesManager(".")

		filename := args[0]
		backup, err := notesManager.LoadBackupFromFile(filename)
		if err != nil {
			return fmt.Errorf("failed to load backup file: %w", err)
		}

		fmt.Printf("📄 Loaded backup from %s (%d notes, created %s)\n",
			filename, len(backup.Notes), backup.BackupTime.Format("2006-01-02 15:04:05"))

		if err := notesManager.RestoreNotesFromBackup(ctx, backup); err != nil {
			return fmt.Errorf("failed to restore notes: %w", err)
		}

		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all commits with conversation notes",
	Long:  `Shows all commits that have conversation notes attached.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		notesManager := notes.NewNotesManager(".")

		backup, err := notesManager.BackupAllNotes(ctx)
		if err != nil {
			return fmt.Errorf("failed to list notes: %w", err)
		}

		if len(backup.Notes) == 0 {
			fmt.Println("No conversation notes found.")
			return nil
		}

		fmt.Printf("Found %d commits with conversation notes:\n\n", len(backup.Notes))
		for commitHash, note := range backup.Notes {
			// Get commit subject
			fmt.Printf("• %s (%s)\n", commitHash[:8], note.Timestamp.Format("2006-01-02 15:04"))
			fmt.Printf("  Session: %s\n", note.SessionID)
			fmt.Printf("  Tools: %v\n\n", note.ToolsUsed)
		}

		fmt.Printf("💡 View notes with: git notes --ref=claude-conversations show <commit>\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(notesCmd)
	notesCmd.AddCommand(backupCmd)
	notesCmd.AddCommand(restoreCmd)
	notesCmd.AddCommand(listCmd)
}
