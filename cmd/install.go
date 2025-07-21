package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/imjasonh/cnotes/internal/config"
	"github.com/spf13/cobra"
)

var (
	uninstall  bool
	global     bool
	local      bool
	installCmd = &cobra.Command{
		Use:   "install",
		Short: "Install cnotes to capture git conversation notes",
		Long: `Install cnotes as a Claude Code hook handler to automatically capture conversation context in git notes.

By default, installs to project settings (.claude/settings.json in current directory).
Use --global for user settings (~/.claude/settings.json).
Use --local for local directory settings (./.claude/settings.json).

This command will:
1. Find or create the appropriate settings.json file
2. Add cnotes to handle PostToolUse events for Bash commands
3. Configure git to preserve notes during rebases

Use --uninstall to remove cnotes from Claude settings.`,
		RunE: runInstall,
	}
)

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVar(&uninstall, "uninstall", false, "Remove hooks from Claude settings")
	installCmd.Flags().BoolVar(&global, "global", false, "Install to global settings (~/.claude/settings.json)")
	installCmd.Flags().BoolVar(&local, "local", false, "Install to local settings (./.claude/settings.json)")
	installCmd.MarkFlagsMutuallyExclusive("global", "local")
}

func runInstall(cmd *cobra.Command, args []string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	executable, err = filepath.Abs(executable)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Determine settings path based on flags
	var settingsPath string
	var scope string
	if global {
		settingsPath = config.GetGlobalSettingsPath()
		scope = "global"
	} else if local {
		settingsPath = config.GetLocalSettingsPath()
		scope = "local"
	} else {
		settingsPath = config.GetProjectSettingsPath()
		scope = "project"
	}

	if uninstall {
		slog.Info("uninstalling hooks", "binary", executable, "scope", scope)
		if err := config.UninstallHooksFromPath(executable, settingsPath); err != nil {
			return fmt.Errorf("failed to uninstall hooks: %w", err)
		}
		fmt.Printf("✓ cnotes uninstalled successfully from %s settings\n", scope)
		return nil
	}

	slog.Info("installing hooks", "binary", executable, "scope", scope)
	if err := config.InstallHooksToPath(executable, settingsPath); err != nil {
		return fmt.Errorf("failed to install hooks: %w", err)
	}

	fmt.Printf("✓ cnotes installed successfully to %s settings\n", scope)
	fmt.Printf("  Binary: %s\n", executable)
	fmt.Printf("  Settings: %s\n", settingsPath)
	fmt.Printf("\nWhat cnotes does:\n")
	fmt.Println("  • Monitors git commit commands executed through Claude")
	fmt.Println("  • Automatically captures conversation context in git notes")
	fmt.Println("  • Includes user prompts and tool interactions since last commit")
	fmt.Println("  • Scans all transcript files in the project for cross-session context")
	fmt.Printf("\nGit notes configuration:\n")
	fmt.Printf("  • Notes ref: claude-conversations\n")
	fmt.Printf("  • Use 'cnotes show' to view conversation notes for commits\n")
	fmt.Printf("  • Use 'cnotes list' to see all commits with notes\n")
	fmt.Printf("  • Use 'cnotes backup/restore' to manage your notes\n")

	return nil
}
