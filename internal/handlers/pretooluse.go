package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/imjasonh/hooks/internal/hooks"
)

type DangerousPattern struct {
	Pattern     *regexp.Regexp
	Description string
}

var dangerousPatterns = []DangerousPattern{
	{regexp.MustCompile(`rm\s+-rf\s+/[^a-zA-Z]`), "recursive deletion of root filesystem"},
	{regexp.MustCompile(`rm\s+-rf\s+/\*`), "recursive deletion of root filesystem contents"},
	{regexp.MustCompile(`:\(\)\{\s*:\|\:&\s*\};\:`), "fork bomb"},
	{regexp.MustCompile(`mkfs\.`), "filesystem formatting"},
	{regexp.MustCompile(`dd\s+if=/dev/zero`), "disk wiping with dd"},
	{regexp.MustCompile(`>\s*/dev/sd[a-z]`), "writing directly to disk device"},
	{regexp.MustCompile(`wget\s+https?://`), "downloading files from internet"},
	{regexp.MustCompile(`curl\s+https?://`), "downloading files from internet"},
	{regexp.MustCompile(`chmod\s+\+x.*\.(sh|py|pl).*&&.*\./`), "download and execute pattern"},
	{regexp.MustCompile(`sudo\s+rm\s+-rf`), "privileged recursive deletion"},
	{regexp.MustCompile(`>/etc/passwd`), "overwriting system password file"},
	{regexp.MustCompile(`>/etc/shadow`), "overwriting system shadow file"},
}

func ValidateBashCommand(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	bashInput, err := input.GetBashInput()
	if err != nil {
		slog.Debug("no bash input found", "error", err)
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	if bashInput.Command == "" {
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	// Check against dangerous patterns
	for _, pattern := range dangerousPatterns {
		if pattern.Pattern.MatchString(bashInput.Command) {
			slog.Warn("blocked dangerous command",
				"command", bashInput.Command,
				"reason", pattern.Description)
			return hooks.HookOutput{
				Decision: "block",
				Reason:   fmt.Sprintf("Command blocked: %s", pattern.Description),
			}, nil
		}
	}

	// Modify sudo commands to be non-interactive
	if strings.Contains(bashInput.Command, "sudo") && !strings.Contains(bashInput.Command, "sudo -n") {
		slog.Info("modifying sudo command to non-interactive")
		return hooks.HookOutput{
			Decision: "approve",
			ModifiedParameters: map[string]any{
				"command": strings.ReplaceAll(bashInput.Command, "sudo", "sudo -n"),
			},
		}, nil
	}

	return hooks.HookOutput{Decision: "approve"}, nil
}

var sensitiveFiles = []string{
	".env",
	".aws/credentials",
	".ssh/id_rsa",
	"secrets",
	"password",
	"token",
	"key.pem",
}

func PreventSensitiveFileEdits(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	fileInput, err := input.GetFileInput()
	if err != nil {
		slog.Debug("no file input found", "error", err)
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	if fileInput.FilePath == "" {
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	lowerPath := strings.ToLower(fileInput.FilePath)
	for _, sensitive := range sensitiveFiles {
		if strings.Contains(lowerPath, strings.ToLower(sensitive)) {
			slog.Warn("blocked sensitive file edit",
				"file", fileInput.FilePath,
				"pattern", sensitive)
			return hooks.HookOutput{
				Decision: "block",
				Reason:   fmt.Sprintf("Cannot edit sensitive file: %s", fileInput.FilePath),
			}, nil
		}
	}

	return hooks.HookOutput{Decision: "approve"}, nil
}
