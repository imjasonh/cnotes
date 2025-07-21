package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	debug   bool
	rootCmd = &cobra.Command{
		Use:   "hooks",
		Short: "Claude Code hooks runner",
		Long: `A Go binary that handles Claude Code hooks for validating and modifying tool usage.

This tool makes it easy to write custom hooks as simple Go functions and automatically
register them to run when Claude Code executes various tools.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug {
				opts := &slog.HandlerOptions{Level: slog.LevelDebug}
				handler := slog.NewTextHandler(os.Stderr, opts)
				slog.SetDefault(slog.New(handler))
			}
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
}