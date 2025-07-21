package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"

	"github.com/imjasonh/hooks/internal/hooks"
)

func init() {
	hooks.RegisterHook(hooks.EventNotification, "*", SpeakNotification)
}

func SpeakNotification(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	// Only run on macOS
	if runtime.GOOS != "darwin" {
		return hooks.HookOutput{Decision: "continue"}, nil
	}

	// Check if say command is available
	if _, err := exec.LookPath("say"); err != nil {
		slog.Debug("say command not found", "error", err)
		return hooks.HookOutput{Decision: "continue"}, nil
	}

	// Build the message to speak
	var message string
	if input.Notification.Permission != "" {
		// Permission request
		message = fmt.Sprintf("Claude is requesting permission to use %s. %s", 
			input.Notification.Tool, 
			input.Notification.Message)
	} else {
		// Regular notification
		message = fmt.Sprintf("Notification from %s: %s",
			input.Notification.Tool,
			input.Notification.Message)
	}

	// Sanitize the message for speech
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\"", "'")

	// Run say command with a shorter message if it's too long
	if len(message) > 200 {
		message = message[:197] + "..."
	}

	cmd := exec.CommandContext(ctx, "say", "-v", "Samantha", message)
	if err := cmd.Start(); err != nil {
		slog.Error("failed to start say command", "error", err)
		return hooks.HookOutput{Decision: "continue"}, nil
	}

	// Don't wait for completion to avoid blocking
	go func() {
		if err := cmd.Wait(); err != nil {
			slog.Debug("say command failed", "error", err)
		}
	}()

	slog.Info("speaking notification",
		"tool", input.Notification.Tool,
		"permission", input.Notification.Permission != "",
		"message_length", len(message))

	return hooks.HookOutput{Decision: "continue"}, nil
}