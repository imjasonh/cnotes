package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/imjasonh/hooks/internal/config"
)

var (
	uninstall bool
	installCmd = &cobra.Command{
		Use:   "install",
		Short: "Install hooks to Claude settings",
		Long: `Install this binary as a hook handler in your Claude settings.

This command will:
1. Find or create ~/.claude/settings.json
2. Add this binary to handle all hook events
3. Configure it to match all tools

Use --uninstall to remove the hooks.`,
		RunE: runInstall,
	}
)

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVar(&uninstall, "uninstall", false, "Remove hooks from Claude settings")
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

	if uninstall {
		slog.Info("uninstalling hooks", "binary", executable)
		if err := config.UninstallHooks(executable); err != nil {
			return fmt.Errorf("failed to uninstall hooks: %w", err)
		}
		fmt.Println("✓ Hooks uninstalled successfully")
		return nil
	}

	slog.Info("installing hooks", "binary", executable)
	if err := config.InstallHooks(executable); err != nil {
		return fmt.Errorf("failed to install hooks: %w", err)
	}

	settingsPath := config.GetSettingsPath()
	fmt.Printf("✓ Hooks installed successfully\n")
	fmt.Printf("  Binary: %s\n", executable)
	fmt.Printf("  Settings: %s\n", settingsPath)
	fmt.Printf("\nThe following hooks are now active:\n")
	fmt.Println("  • pre_tool_use: Validates bash commands and prevents sensitive file edits")
	fmt.Println("  • post_tool_use: Logs tool usage and runs goimports on modified Go files")
	fmt.Println("  • user_prompt_submit: Adds project context to prompts")
	fmt.Println("\nTo test: echo '{\"event\":\"pre_tool_use\",\"tool\":\"Bash\",\"tool_use_request\":{\"tool\":\"Bash\",\"parameters\":{\"command\":\"ls\"}}}' | hooks run")

	return nil
}