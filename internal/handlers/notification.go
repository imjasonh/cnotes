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

	// Build the notification content
	var title, message, subtitle string
	if input.Notification.Permission != "" {
		// Permission request
		title = "Claude Permission Request"
		subtitle = fmt.Sprintf("Tool: %s", input.Notification.Tool)
		message = fmt.Sprintf("Requesting permission to %s", input.Notification.Message)
	} else {
		// Regular notification
		title = "Claude Notification"
		subtitle = fmt.Sprintf("Tool: %s", input.Notification.Tool)
		message = input.Notification.Message
	}

	// Show notification using terminal-notifier if available
	if _, err := exec.LookPath("terminal-notifier"); err == nil {
		args := []string{
			"-title", title,
			"-subtitle", subtitle,
			"-message", message,
			"-sound", "default",
			"-group", "claude-hooks",
		}
		
		cmd := exec.CommandContext(ctx, "terminal-notifier", args...)
		if err := cmd.Start(); err != nil {
			slog.Error("failed to show notification", "error", err)
		} else {
			// Don't wait for completion
			go func() {
				if err := cmd.Wait(); err != nil {
					slog.Debug("terminal-notifier failed", "error", err)
				}
			}()
			slog.Info("showed notification",
				"tool", input.Notification.Tool,
				"permission", input.Notification.Permission != "")
		}
	} else {
		slog.Debug("terminal-notifier not found, install with: brew install terminal-notifier")
	}

	// Also speak the notification if say is available
	if _, err := exec.LookPath("say"); err == nil {
		spokenMessage := fmt.Sprintf("%s. %s", subtitle, message)
		
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
		}
	}

	return hooks.HookOutput{Decision: "continue"}, nil
}