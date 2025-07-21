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
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	// Build the notification content for speech
	var message string

	// Check if we have the direct message format (newer)
	if input.Message != "" {
		message = input.Message
	} else if input.Notification.Permission != "" {
		// Permission request (older format)
		message = fmt.Sprintf("Requesting permission to %s", input.Notification.Message)
	} else if input.Notification.Message != "" {
		// Regular notification (older format)
		message = input.Notification.Message
	} else {
		// No valid notification content
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	// Speak the notification using macOS say command
	if _, err := exec.LookPath("say"); err == nil {
		// Build more informative spoken message
		var spokenMessage string
		if input.Notification.Permission != "" {
			spokenMessage = fmt.Sprintf("Claude is requesting permission for %s: %s", input.Notification.Tool, message)
		} else if input.Notification.Tool != "" {
			spokenMessage = fmt.Sprintf("Claude notification from %s: %s", input.Notification.Tool, message)
		} else {
			spokenMessage = fmt.Sprintf("Claude notification: %s", message)
		}

		// Sanitize for speech
		spokenMessage = strings.ReplaceAll(spokenMessage, "\n", " ")
		spokenMessage = strings.ReplaceAll(spokenMessage, "\"", "'")

		// Truncate if too long
		if len(spokenMessage) > 200 {
			spokenMessage = spokenMessage[:197] + "..."
		}

		cmd := exec.CommandContext(ctx, "say", "-v", "Samantha", spokenMessage)
		if err := cmd.Start(); err != nil {
			slog.Debug("failed to start say command", "error", err)
		} else {
			// Don't wait for completion to avoid blocking
			go func() {
				if err := cmd.Wait(); err != nil {
					slog.Debug("say command failed", "error", err)
				}
			}()
			slog.Info("spoke notification",
				"tool", input.Notification.Tool,
				"permission", input.Notification.Permission != "")
		}
	}

	return hooks.HookOutput{Decision: "approve"}, nil
}
