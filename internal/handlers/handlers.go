// Package handlers provides hook implementations for all Claude Code events.
//
// The following hook events are implemented:
// - PreToolUse: Command validation and sensitive file protection (pretooluse.go)
// - PostToolUse: Tool usage logging and goimports automation (posttooluse.go, goimports.go)
// - UserPromptSubmit: Project context injection (userprompt.go)
// - Notification: System notification logging (notification.go)
// - Stop: Session completion logging (stop.go)
// - SubagentStop: Subagent completion logging (subagentstop.go)
// - PreCompact: Context compaction handling (precompact.go)
//
// Each handler file registers its hooks in init() functions that are automatically
// called when the package is imported.
package handlers

import (
	"github.com/imjasonh/hooks/internal/hooks"
)

func init() {
	// PreToolUse hooks - validation and security
	hooks.RegisterHook(hooks.EventPreToolUse, "Bash", ValidateBashCommand)
	hooks.RegisterHook(hooks.EventPreToolUse, "Write|Edit|MultiEdit", PreventSensitiveFileEdits)

	// PostToolUse hooks - logging and automation
	hooks.RegisterHook(hooks.EventPostToolUse, "*", LogToolUsage)

	// UserPromptSubmit hooks - context enhancement
	hooks.RegisterHook(hooks.EventUserPromptSubmit, "*", AddProjectContext)
}
