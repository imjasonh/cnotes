package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/imjasonh/hooks/internal/hooks"
	_ "github.com/imjasonh/hooks/internal/handlers"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run hook handler (called by Claude Code)",
	Long: `Run the hook handler by reading JSON from stdin and processing it.

This command is typically called automatically by Claude Code when hooks are triggered.
You can also use it for testing by piping JSON input.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		code := hooks.RunExitCode(ctx)
		if code != 0 {
			os.Exit(code)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}