package handlers

import (
	"github.com/imjasonh/hooks/internal/hooks"
)

func init() {
	hooks.RegisterHook(hooks.EventPreToolUse, "Bash", ValidateBashCommand)
	hooks.RegisterHook(hooks.EventPreToolUse, "Write|Edit|MultiEdit", PreventSensitiveFileEdits)
	
	hooks.RegisterHook(hooks.EventPostToolUse, "*", LogToolUsage)
	
	hooks.RegisterHook(hooks.EventUserPromptSubmit, "*", AddProjectContext)
}