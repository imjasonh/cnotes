package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/imjasonh/cnotes/internal/config"
	"github.com/imjasonh/cnotes/internal/notes"
	"github.com/spf13/cobra"
)

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

var showCmd = &cobra.Command{
	Use:   "show [commit]",
	Short: "Show conversation notes for a commit in Markdown format",
	Long: `Pretty-prints the conversation context for a commit in readable Markdown format.
If no commit is specified, shows notes for HEAD.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		notesManager := notes.NewNotesManager(".")

		// Default to HEAD if no commit specified
		commit := "HEAD"
		if len(args) > 0 {
			commit = args[0]
		}

		// Get the conversation note
		note, err := notesManager.GetConversationNote(ctx, commit)
		if err != nil {
			return fmt.Errorf("failed to get conversation note: %w", err)
		}

		if note == nil {
			fmt.Printf("No conversation notes found for commit %s\n", commit)
			fmt.Printf("💡 Use 'cnotes notes list' to see which commits have notes\n")
			return nil
		}

		// Pretty-print in Markdown format
		cfg := config.LoadNotesConfig(".")
		printConversationMarkdown(*note, commit, cfg)
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

		fmt.Printf("💡 View notes with: 'cnotes notes show <commit>'\n")
		return nil
	},
}

// printConversationMarkdown formats a conversation note as readable Markdown
func printConversationMarkdown(note notes.ConversationNote, commit string, cfg *config.NotesConfig) {
	fmt.Printf("# Claude Conversation Notes\n\n")

	// Get commit info
	if commitInfo := getCommitInfo(commit); commitInfo != "" {
		fmt.Printf("**Commit:** `%s`\n", commitInfo)
	}

	fmt.Printf("**Session ID:** `%s`\n", note.SessionID)
	fmt.Printf("**Timestamp:** %s\n", note.Timestamp.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("**Claude Version:** %s\n", note.ClaudeVersion)
	fmt.Printf("**Tools Used:** %s\n\n", strings.Join(note.ToolsUsed, ", "))

	// Conversation transcript
	if note.ConversationExcerpt != "" {
		fmt.Printf("## Conversation Transcript\n\n")
		// Clean up and format the conversation excerpt for better readability
		formatted := formatConversationExcerpt(note.ConversationExcerpt, cfg)
		fmt.Printf("%s\n\n", formatted)
	}

	fmt.Printf("---\n")
	fmt.Printf("💡 *Generated by `cnotes`*\n")
}

// formatConversationExcerpt cleans up the conversation excerpt for better readability
func formatConversationExcerpt(excerpt string, cfg *config.NotesConfig) string {
	// Replace escaped newlines with actual newlines
	formatted := strings.ReplaceAll(excerpt, "\\n", "\n")

	// Replace escaped quotes with regular quotes
	formatted = strings.ReplaceAll(formatted, "\\\"", "\"")

	// Replace escaped backslashes
	formatted = strings.ReplaceAll(formatted, "\\\\", "\\")

	// Clean up common JSON escape sequences
	formatted = strings.ReplaceAll(formatted, "\\t", "    ") // tabs to spaces
	formatted = strings.ReplaceAll(formatted, "\\r", "")     // remove carriage returns

	// Fix any double newlines that might have been created
	for strings.Contains(formatted, "\n\n\n") {
		formatted = strings.ReplaceAll(formatted, "\n\n\n", "\n\n")
	}

	// Format different message types for better visual distinction
	lines := strings.Split(formatted, "\n")
	var formattedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			formattedLines = append(formattedLines, "")
			continue
		}

		switch {
		case strings.HasPrefix(line, "User:") || (cfg != nil && cfg.UserEmoji != "" && strings.HasPrefix(line, cfg.UserEmoji)):
			// Bold user prompts
			formattedLines = append(formattedLines, "**"+line+"**")

		case strings.HasPrefix(line, "Claude:") || (cfg != nil && cfg.AssistantEmoji != "" && strings.HasPrefix(line, cfg.AssistantEmoji)):
			// Keep Claude responses as-is
			formattedLines = append(formattedLines, line)

		case strings.HasPrefix(line, "Tool ("):
			// Format tool uses in code blocks
			toolParts := strings.SplitN(line, ": ", 2)
			if len(toolParts) == 2 {
				formattedLines = append(formattedLines, toolParts[0]+":")
				formattedLines = append(formattedLines, "```")
				formattedLines = append(formattedLines, toolParts[1])
				formattedLines = append(formattedLines, "```")
			} else {
				formattedLines = append(formattedLines, line)
			}

		case strings.HasPrefix(line, "Result:"):
			// Format results in code blocks
			resultParts := strings.SplitN(line, ": ", 2)
			if len(resultParts) == 2 {
				formattedLines = append(formattedLines, "_"+resultParts[0]+":_")
				formattedLines = append(formattedLines, "```")
				formattedLines = append(formattedLines, resultParts[1])
				formattedLines = append(formattedLines, "```")
			} else {
				formattedLines = append(formattedLines, line)
			}

		default:
			// Handle multi-line content that might be part of a previous block
			formattedLines = append(formattedLines, line)
		}
	}

	return strings.Join(formattedLines, "\n")
}

// getCommitInfo returns formatted commit information
func getCommitInfo(commit string) string {
	cmd := exec.Command("git", "log", "--oneline", "-1", commit)
	output, err := cmd.Output()
	if err != nil {
		return commit
	}
	return strings.TrimSpace(string(output))
}

func init() {
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(listCmd)
}
