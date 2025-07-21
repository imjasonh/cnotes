# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

```bash
# Build the binary
go build -o hooks

# Install hooks to Claude settings (project-level by default)
./hooks install

# Install to different scopes
./hooks install --global    # ~/.claude/settings.json
./hooks install --local     # ./.claude/settings.local.json

# Uninstall hooks
./hooks install --uninstall

# Test hook execution manually (for debugging)
echo '{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":"{\"command\":\"ls\"}"}' | ./hooks run

# Run with debug logging
./hooks run --debug
```

## Architecture Overview

This is a Claude Code hooks system that intercepts and processes all tool usage events. The architecture follows a registry pattern where individual hook handlers register themselves for specific events and tools.

### Core Components

**`internal/hooks/`** - Core hook engine
- `types.go` - Defines all hook input/output types and helper methods. Uses `json.RawMessage` for parameters to avoid type casting
- `registry.go` - Central registry for mapping events+tools to handler functions  
- `runner.go` - Main execution engine that processes hook chains and handles JSON I/O

**`internal/handlers/`** - Hook implementations
- Each file implements handlers for specific events (pretooluse.go, posttooluse.go, etc.)
- Handlers auto-register in `init()` functions when package is imported
- All 7 Claude Code events are implemented: PreToolUse, PostToolUse, UserPromptSubmit, Notification, Stop, SubagentStop, PreCompact

**`internal/config/`** - Claude settings management  
- `settings.go` - Handles reading/writing Claude's settings.json files with support for global, local, and project-level configurations

**`cmd/`** - CLI interface using Cobra
- `run.go` - Handles hook execution (called by Claude Code)
- `install.go` - Manages hook installation/uninstallation

### Type Safety and JSON Handling

The codebase avoids string type casting entirely by:
1. Using `json.RawMessage` for all parameter storage
2. Providing typed structs (`BashToolInput`, `FileToolInput`) for tool-specific data
3. Using `input.GetBashInput()`, `input.GetFileInput()` helper methods that unmarshal JSON directly to typed structs
4. Never using `.(string)` type assertions anywhere in the codebase

### Hook Registration Pattern

```go
func init() {
    hooks.RegisterHook(hooks.EventPreToolUse, "Bash", ValidateBashCommand)
    hooks.RegisterHook(hooks.EventPreToolUse, "Write|Edit|MultiEdit", PreventSensitiveFileEdits) 
}
```

Matchers support:
- Exact tool names: `"Bash"`
- Regex patterns: `"Write|Edit|MultiEdit"`  
- Wildcard: `"*"` for all tools

### Hook Function Signature

All hook functions follow this signature:
```go
func HookName(ctx context.Context, input hooks.HookInput) (hooks.HookOutput, error)
```

Hook outputs can:
- Block execution: `Decision: "block"`  
- Allow with modifications: `Decision: "approve"` + `ModifiedParameters`
- Add context to responses: `AdditionalContext`
- Modify user prompts: `ModifiedUserPrompt`

### Settings File Structure

The system uses Claude's native hook configuration format with proper event name mapping:
```json
{
  "hooks": {
    "PreToolUse": [{"matcher": ".*", "hooks": [{"type": "command", "command": "/path/to/hooks run"}]}],
    "PostToolUse": [{"matcher": ".*", "hooks": [{"type": "command", "command": "/path/to/hooks run"}]}]
  }
}
```

## Key Design Principles

- **No Type Casting**: All JSON handling uses proper unmarshaling to typed structs
- **Auto-Registration**: Hook handlers register themselves via `init()` functions  
- **Comprehensive Coverage**: All 7 Claude Code events are implemented with placeholder functionality
- **Project-First**: Defaults to project-level `.claude/settings.json` installation
- **Extensible**: Easy to add new hook handlers by creating new files in `internal/handlers/`

When adding new hooks, create a new file in `internal/handlers/`, implement the hook function, and register it in an `init()` function. The handler will be automatically loaded when the package is imported.