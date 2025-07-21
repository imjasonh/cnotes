package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/imjasonh/hooks/internal/hooks"
)

func init() {
	hooks.RegisterHook(hooks.EventPreToolUse, "Bash", WarnAboutGitNotesLoss)
}

// WarnAboutGitNotesLoss warns users about potential git notes loss during destructive operations
func WarnAboutGitNotesLoss(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	bashInput, err := input.GetBashInput()
	if err != nil {
		return hooks.HookOutput{Decision: "approve"}, nil
	}

	command := strings.TrimSpace(bashInput.Command)

	// Check for destructive git operations that might lose notes
	destructiveOperations := []struct {
		pattern string
		warning string
	}{
		{
			pattern: "git rebase",
			warning: "⚠️  Git rebase can lose conversation notes attached to commits. Consider:\n" +
				"   • Run 'git notes --ref=claude-conversations list' to see which commits have notes\n" +
				"   • Use 'git -c notes.rewrite.mode=copy rebase' to preserve notes\n" +
				"   • Or manually backup notes before rebasing with 'git notes --ref=claude-conversations show <commit>'",
		},
		{
			pattern: "git reset --hard",
			warning: "⚠️  Hard reset will lose commits and their conversation notes permanently.\n" +
				"   • Consider using 'git reset --soft' or 'git reset --mixed' instead\n" +
				"   • Backup important notes first with 'git notes --ref=claude-conversations show <commit>'",
		},
		{
			pattern: "git commit --amend",
			warning: "⚠️  Amending commits changes their hash and may lose conversation notes.\n" +
				"   • Notes are attached to the original commit hash\n" +
				"   • Consider creating a new commit instead of amending",
		},
	}

	for _, op := range destructiveOperations {
		if strings.Contains(command, op.pattern) {
			slog.Info("warning user about potential git notes loss",
				"command", command,
				"operation", op.pattern)

			return hooks.HookOutput{
				Decision:          "approve", // Don't block, just warn
				AdditionalContext: fmt.Sprintf("\n%s\n\nDo you want to proceed with this operation?", op.warning),
			}, nil
		}
	}

	return hooks.HookOutput{Decision: "approve"}, nil
}
