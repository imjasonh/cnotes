# Claude Code Hooks

A Go binary that handles Claude Code hooks for validating and modifying tool usage. This tool makes it easy to write custom hooks as simple Go functions.

## Installation

```bash
go build -o hooks
./hooks install
```

This will configure your `~/.claude/settings.json` to use this binary for all hook events.

## Usage

### Writing Custom Hooks

Hooks are simple Go functions that follow this signature:

```go
func MyHook(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
    // Your logic here
    return hooks.HookOutput{Decision: "continue"}, nil
}
```

Register your hook in an init function:

```go
func init() {
    // Register for specific tools
    hooks.RegisterHook(hooks.EventPreToolUse, "Bash", ValidateBashCommand)
    
    // Use regex patterns
    hooks.RegisterHook(hooks.EventPreToolUse, "Write|Edit", PreventSensitiveEdits)
    
    // Match all tools
    hooks.RegisterHook(hooks.EventPostToolUse, "*", LogEverything)
}
```

### Example: Block Dangerous Commands

```go
func BlockRmRf(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
    cmd := input.ToolUseRequest.Parameters["command"].(string)
    if strings.Contains(cmd, "rm -rf /") {
        return hooks.HookOutput{
            Decision: "block",
            Reason:   "Cannot execute rm -rf on root directory",
        }, nil
    }
    return hooks.HookOutput{Decision: "continue"}, nil
}
```

### Example: Add Context to Prompts

```go
func AddGitInfo(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
    branch := getCurrentGitBranch()
    return hooks.HookOutput{
        Decision: "continue",
        AdditionalContext: fmt.Sprintf("Current git branch: %s", branch),
    }, nil
}
```

### Example: Modify Tool Parameters

```go
func ForceNonInteractiveSudo(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error) {
    cmd := input.ToolUseRequest.Parameters["command"].(string)
    if strings.Contains(cmd, "sudo") && !strings.Contains(cmd, "sudo -n") {
        return hooks.HookOutput{
            Decision: "continue",
            ModifiedParameters: map[string]interface{}{
                "command": strings.ReplaceAll(cmd, "sudo", "sudo -n"),
            },
        }, nil
    }
    return hooks.HookOutput{Decision: "continue"}, nil
}
```

## Built-in Hooks

This package includes several example hooks:

### Pre Tool Use
- **Bash Command Validator**: Blocks dangerous commands like `rm -rf /`
- **Sensitive File Protection**: Prevents editing `.env`, `.aws/credentials`, etc.

### Post Tool Use
- **Tool Usage Logger**: Logs all tool executions with structured logging
- **Go Imports**: Automatically runs `goimports -w` on modified Go files

### User Prompt Submit
- **Project Context**: Adds git branch and project type information

### Notification
- **Visual Notifications**: Shows native macOS notifications via `terminal-notifier`
- **Speech Notifications**: Uses macOS 'say' command to speak notifications aloud

Note: Install `terminal-notifier` with `brew install terminal-notifier` for visual notifications.

## Testing Hooks

Test individual hooks by piping JSON:

```bash
echo '{
  "event": "pre_tool_use",
  "tool": "Bash",
  "tool_use_request": {
    "tool": "Bash",
    "parameters": {
      "command": "rm -rf /"
    }
  }
}' | ./hooks run
```

Enable debug logging:

```bash
./hooks run --debug
```

## Hook Events

- `pre_tool_use`: Before any tool execution
- `post_tool_use`: After successful tool execution
- `user_prompt_submit`: Before processing user prompts
- `stop`: When main agent finishes
- `subagent_stop`: When subagent finishes
- `notification`: For permission requests
- `pre_compact`: Before context compaction

## Configuration

Hooks are configured in `~/.claude/settings.json`:

```json
{
  "hooks": [
    {
      "events": ["pre_tool_use", "post_tool_use"],
      "matchers": [".*"],
      "cmds": ["/path/to/hooks run"]
    }
  ]
}
```

## Security Warning

⚠️ **USE AT YOUR OWN RISK**: Hooks execute automatically when Claude Code runs tools. Ensure your hooks are thoroughly tested and secure.

## Development

To add new hooks:

1. Create a new file in `internal/handlers/`
2. Write your hook function
3. Register it in an `init()` function
4. Rebuild and reinstall: `go build && ./hooks install`

## Uninstalling

```bash
./hooks install --uninstall
```

This removes all hook configurations from your Claude settings.