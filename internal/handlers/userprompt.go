package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/imjasonh/hooks/internal/hooks"
)

func AddProjectContext(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
	// Speak notification for hook event on macOS
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("say"); err == nil {
			// Extract first few words from prompt for context
			promptPreview := input.Prompt
			if len(promptPreview) > 50 {
				promptPreview = promptPreview[:47] + "..."
			}
			// Remove newlines for speech
			promptPreview = strings.ReplaceAll(promptPreview, "\n", " ")

			spokenMessage := fmt.Sprintf("User prompt received: %s", promptPreview)
			cmd := exec.CommandContext(ctx, "say", "-v", "Samantha", spokenMessage)
			if err := cmd.Start(); err == nil {
				go func() {
					if err := cmd.Wait(); err != nil {
						slog.Debug("say command failed", "error", err)
					}
				}()
			}
		}
	}

	var contexts []string

	if gitRoot := findGitRoot(input.CWD); gitRoot != "" {
		contexts = append(contexts, fmt.Sprintf("Git repository root: %s", gitRoot))

		if branch := getCurrentBranch(gitRoot); branch != "" {
			contexts = append(contexts, fmt.Sprintf("Current branch: %s", branch))
		}
	}

	if _, err := os.Stat(filepath.Join(input.CWD, "go.mod")); err == nil {
		contexts = append(contexts, "Project type: Go module")
	} else if _, err := os.Stat(filepath.Join(input.CWD, "package.json")); err == nil {
		contexts = append(contexts, "Project type: Node.js/npm")
	} else if _, err := os.Stat(filepath.Join(input.CWD, "Cargo.toml")); err == nil {
		contexts = append(contexts, "Project type: Rust/Cargo")
	}

	if len(contexts) > 0 {
		additionalContext := "Project context:\n" + strings.Join(contexts, "\n")
		slog.Info("adding project context", "context_lines", len(contexts))
		return hooks.HookOutput{
			Decision:          "approve",
			AdditionalContext: additionalContext,
		}, nil
	}

	return hooks.HookOutput{Decision: "approve"}, nil
}

func findGitRoot(dir string) string {
	current := dir
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func getCurrentBranch(gitRoot string) string {
	headPath := filepath.Join(gitRoot, ".git", "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}
	ref := strings.TrimSpace(string(content))
	if strings.HasPrefix(ref, "ref: refs/heads/") {
		return strings.TrimPrefix(ref, "ref: refs/heads/")
	}
	return ""
}
