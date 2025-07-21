package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/imjasonh/hooks/internal/hooks"
)

func init() {
	hooks.RegisterHook(hooks.EventNotification, "*", LogNotification)
}

// LogNotification logs notification events for debugging and monitoring.
// This hook is triggered when Claude needs permission to use a tool or when
// prompt input has been idle for 60+ seconds.
//
// Common use cases:
// - Debug notification flow
// - Monitor permission requests
// - Track user idle states
// - Audit notification patterns
func LogNotification(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	var message string
	var notificationType string

	// Check if we have the direct message format (newer)
	if input.Message != "" {
		message = input.Message
		notificationType = "direct"
	} else if input.Notification.Permission != "" {
		// Permission request (older format)
		message = input.Notification.Message
		notificationType = "permission"
	} else if input.Notification.Message != "" {
		// Regular notification (older format)
		message = input.Notification.Message
		notificationType = "message"
	} else {
		// No valid notification content
		slog.Debug("notification hook triggered with no message content", "session_id", input.SessionID)
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	slog.Info("notification received",
		"type", notificationType,
		"message", message,
		"tool", input.Notification.Tool,
		"permission", input.Notification.Permission != "",
		"session_id", input.SessionID,
		"timestamp", time.Now().Unix())

	return hooks.HookOutput{Decision: "approve"}, nil
}
