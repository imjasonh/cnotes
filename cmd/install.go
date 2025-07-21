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
		fmt.Printf("✓ Hooks uninstalled successfully from %s settings\n", scope)
		return nil
	}

	slog.Info("installing hooks", "binary", executable, "scope", scope)
	if err := config.InstallHooksToPath(executable, settingsPath); err != nil {
		return fmt.Errorf("failed to install hooks: %w", err)
	}

	fmt.Printf("✓ Hooks installed successfully to %s settings\n", scope)
	fmt.Printf("  Binary: %s\n", executable)
	fmt.Printf("  Settings: %s\n", settingsPath)
	fmt.Printf("\nThe following hooks are now active:\n")
	fmt.Println("  • pre_tool_use: Validates bash commands and prevents sensitive file edits")
	fmt.Println("  • post_tool_use: Logs tool usage and runs goimports on modified Go files")
	fmt.Println("  • user_prompt_submit: Adds project context to prompts")
	fmt.Println("  • notification: Logs notification events")
	fmt.Println("  • stop: Logs session completion")
	fmt.Println("  • subagent_stop: Logs subagent completion")
	fmt.Println("  • pre_compact: Handles context compaction events")
	fmt.Println("\nTo test: echo '{\"event\":\"pre_tool_use\",\"tool\":\"Bash\",\"tool_use_request\":{\"tool\":\"Bash\",\"parameters\":{\"command\":\"ls\"}}}' | hooks run")

	return nil
}
