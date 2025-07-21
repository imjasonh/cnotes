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
    bashInput, err := input.GetBashInput()
    if err != nil {
        return hooks.HookOutput{Decision: "approve"}, nil
    }
    
    if strings.Contains(bashInput.Command, "rm -rf /") {
        return hooks.HookOutput{
            Decision: "block",
            Reason:   "Cannot execute rm -rf on root directory",
        }, nil
    }
    return hooks.HookOutput{Decision: "approve"}, nil
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
    bashInput, err := input.GetBashInput()
    if err != nil {
        return hooks.HookOutput{Decision: "approve"}, nil
    }
    
    if strings.Contains(bashInput.Command, "sudo") && !strings.Contains(bashInput.Command, "sudo -n") {
        return hooks.HookOutput{
            Decision: "approve",
            ModifiedParameters: map[string]interface{}{
                "command": strings.ReplaceAll(bashInput.Command, "sudo", "sudo -n"),
            },
        }, nil
    }
    return hooks.HookOutput{Decision: "approve"}, nil
}
```

## Built-in Hooks

This package includes comprehensive hook implementations for all Claude Code events:

### Pre Tool Use
- **Bash Command Validator**: Blocks dangerous commands using regex patterns
- **Sensitive File Protection**: Prevents editing `.env`, `.aws/credentials`, etc.

### Post Tool Use
- **Tool Usage Logger**: Logs all tool executions with structured logging
- **Go Imports**: Automatically runs `goimports -w` on modified Go files

### User Prompt Submit
- **Project Context**: Adds git branch and project type information

### Notification
- **System Notifications**: Logs notification events for debugging and monitoring

### Stop
- **Session Completion**: Logs when main Claude agent finishes responding

### SubagentStop  
- **Subagent Completion**: Logs when Task tool subagents finish responding

### PreCompact
- **Context Compaction**: Handles before context window compaction (manual/auto)

## Git Notes Integration

This hooks system automatically attaches Claude conversation context to git commits using [git notes](https://git-scm.com/docs/git-notes). When you run `git commit` commands, the system captures relevant conversation context and stores it alongside your commits.

### What Gets Stored

Each git note contains structured JSON data with:
- **Session ID**: Unique identifier for the Claude session
- **Timestamp**: When the commit was made
- **Conversation Excerpt**: Recent user prompts and tool interactions
- **Tools Used**: List of tools used during the conversation
- **Commit Context**: Details about the git command and output
- **Claude Version**: Which version of Claude created the commit

### Example Usage

Let's say you're working on a project and ask Claude to implement a feature:

```bash
# You: "Add user authentication to the login form"
# Claude uses various tools (Edit, Write, Bash) to implement the feature
# Then creates a commit:

git commit -m "Add user authentication with password validation

- Implement bcrypt password hashing
- Add login form validation
- Create user session management
- Add error handling for failed logins"
```

The hook automatically captures the conversation context and attaches it to the commit.

### Viewing Conversation Context

To view the conversation context attached to any commit:

```bash
# View notes for the latest commit
git notes --ref=claude-conversations show HEAD

# View notes for a specific commit
git notes --ref=claude-conversations show abc1234

# View notes in git log (one-liner)
git log --show-notes=claude-conversations --oneline

# View detailed notes in git log
git log --show-notes=claude-conversations -1
```

### Example Output

```json
{
  "session_id": "claude_session_20250121_143022",
  "timestamp": "2025-01-21T14:30:45Z",
  "conversation_excerpt": "Recent user prompts:\n- Add user authentication to the login form\n- Make sure to use bcrypt for password hashing\n\nTool interactions:\n- Edit: components/LoginForm.jsx\n- Write: utils/auth.js\n- Bash: npm install bcrypt",
  "tools_used": ["Edit", "Write", "Bash"],
  "commit_context": "Git command: git commit -m 'Add user authentication with password validation'\nResult: [main abc1234] Add user authentication with password validation",
  "claude_version": "claude-sonnet-4-20250514"
}
```

### Configuration

Customize the git notes behavior by creating `.claude/notes.json`:

```json
{
  "enabled": true,
  "max_excerpt_length": 5000,
  "max_prompts": 2,
  "include_tool_output": false,
  "notes_ref": "claude-conversations",
  "exclude_patterns": ["password", "token", "key", "secret"]
}
```

### Privacy Controls

The system includes built-in privacy protections:
- Automatically filters sensitive patterns (passwords, tokens, keys)
- Limits excerpt length to prevent excessive data storage
- Only includes conversation context from the current session
- Configurable exclusion patterns
- Option to disable entirely (`"enabled": false`)

### Sharing Git Notes

Git notes are stored locally by default. To share them with your team:

```bash
# Push notes to remote repository
git push origin refs/notes/claude-conversations

# Pull notes from remote repository
git fetch origin refs/notes/claude-conversations:refs/notes/claude-conversations

# Configure automatic notes fetching
git config remote.origin.fetch '+refs/notes/*:refs/notes/*'
```

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