# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with the cnotes codebase.

## Build and Run Commands

```bash
# Build the binary
go build -o cnotes

# Install cnotes to capture conversation notes (project-level by default)
cnotes install

# Install to different scopes
cnotes install --global    # ~/.claude/settings.json
cnotes install --local     # ./.claude/settings.json

# Uninstall cnotes
cnotes install --uninstall

# View conversation notes for commits
cnotes show              # Show notes for HEAD
cnotes show abc1234      # Show notes for specific commit
cnotes list             # List all commits with notes

# Backup and restore notes
cnotes backup my-backup.json
cnotes restore my-backup.json

# Test hook execution manually (for debugging)
echo '{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":"{\"command\":\"git commit -m test\"}"}' | cnotes
```

## What cnotes Does

cnotes is a focused tool that automatically captures Claude conversation context in git notes. It integrates with Claude Code as a PostToolUse hook handler and:

1. **Monitors git commit commands** in bash tool executions
2. **Extracts conversation context** from Claude transcript files
3. **Attaches structured notes** to commits using `git notes --ref=claude-conversations`
4. **Provides user-friendly commands** to view, backup, and restore conversation notes

## Architecture Overview

**Core Components:**

**`main.go`** - Entry point that calls cmd.Execute()

**`cmd/root.go`** - Main cnotes command that handles Claude Code hook calls
- Receives JSON input from Claude Code hooks via stdin
- Only processes PostToolUse events for Bash commands containing "git commit"
- Extracts conversation context and attaches it as git notes
- Returns JSON response to Claude Code

**`cmd/notes.go`** - User-facing commands for managing notes
- `show` - Pretty-print conversation notes in Markdown format
- `list` - List all commits with conversation notes  
- `backup` - Create JSON backups of all notes
- `restore` - Restore notes from JSON backup files

**`cmd/install.go`** - Installation command
- Configures Claude Code settings to call cnotes as a hook handler
- Supports project-level, global, and local installations
- Sets up git configuration to preserve notes during rebases

**`internal/notes/`** - Git notes operations
- `git_notes.go` - Core git notes CRUD operations using command-line git
- `backup.go` - Backup and restore functionality for notes

**`internal/context/`** - Conversation context extraction
- `conversation.go` - Parses Claude transcripts to extract relevant conversation context
- Privacy-aware filtering to exclude sensitive information

**`internal/config/`** - Configuration management
- `settings.go` - Handles reading/writing Claude settings.json files
- `notes.go` - Configuration for git notes behavior with privacy controls

## Key Features

**Automatic Git Notes Creation:**
- Detects `git commit` commands in Claude conversations
- Extracts commit hashes from git command output
- Stores conversation context as structured JSON in git notes

**Pretty-Printed Viewing:**
```bash
cnotes show              # Beautifully formatted Markdown output
```

**Backup and Restore:**
```bash
cnotes backup notes.json
cnotes restore notes.json
```

**Privacy Controls:**
- Filters sensitive patterns (passwords, tokens, keys)
- Configurable exclusion patterns
- Length limits on conversation excerpts

## Configuration

Create `.claude/notes.json` to customize behavior:
```json
{
  "enabled": true,
  "max_excerpt_length": 5000,
  "max_prompts": 2,
  "notes_ref": "claude-conversations",
  "exclude_patterns": ["password", "token", "key", "secret"],
  "user_emoji": "🧑",
  "assistant_emoji": "🤖"
}
```

## Hook Integration

cnotes registers as a Claude Code hook handler for:
- **PostToolUse** events on **Bash** tools
- Processes git commit commands to attach conversation context
- Configures git to preserve notes during rebase operations

## Development Notes

- **Focused Scope**: Only handles git notes functionality, removed all other hook handling
- **Type Safety**: Uses json.RawMessage and typed structs, no string casting
- **Privacy-First**: Built-in filtering for sensitive information
- **User Experience**: Prioritizes readable Markdown output over raw JSON
- **Reliability**: Comprehensive backup/restore system for data protection

When making changes, ensure:
1. Git notes functionality remains the primary focus
2. Privacy controls are maintained
3. Backup/restore capabilities work correctly
4. Pretty-printed output remains readable
5. Installation process configures Claude Code correctly
