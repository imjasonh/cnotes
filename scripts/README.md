# Git Notes GitHub API Scripts

This directory contains scripts that demonstrate how to access git notes through the GitHub API using the `gh` CLI tool. These scripts show that while GitHub doesn't display git notes in their web interface (since 2014), the notes data is still fully accessible via their API.

## Prerequisites

- `gh` CLI tool installed and authenticated
- `jq` for JSON processing
- Git notes pushed to GitHub with `git push origin refs/notes/*`

## Scripts

### fetch-github-notes.sh

Fetches a specific git note for a given commit.

**Usage:**
```bash
./scripts/fetch-github-notes.sh [owner/repo] [commit-sha] [notes-ref]
```

**Examples:**
```bash
# Fetch note for current repo
./scripts/fetch-github-notes.sh abc123

# Fetch note from specific repo
./scripts/fetch-github-notes.sh owner/repo abc123

# Fetch note from custom ref (default is claude-conversations)
./scripts/fetch-github-notes.sh owner/repo abc123 commits
```

### fetch-all-github-notes.sh

Lists all commits that have notes and displays their content.

**Usage:**
```bash
./scripts/fetch-all-github-notes.sh [owner/repo] [notes-ref]
```

**Examples:**
```bash
# List all notes in current repo
./scripts/fetch-all-github-notes.sh

# List all notes in specific repo
./scripts/fetch-all-github-notes.sh owner/repo

# List notes from custom ref
./scripts/fetch-all-github-notes.sh owner/repo commits
```

### list-commit-notes.sh

Lists all git notes attached to a specific commit across all notes references. A single commit can have notes in multiple refs (e.g., `refs/notes/commits`, `refs/notes/claude-conversations`, etc.).

**Usage:**
```bash
./scripts/list-commit-notes.sh [owner/repo] <commit-sha>
```

**Examples:**
```bash
# List all notes for a commit in current repo
./scripts/list-commit-notes.sh abc123

# List all notes for a commit in specific repo
./scripts/list-commit-notes.sh owner/repo abc123
```

## How It Works

Git notes are stored in GitHub as regular git objects:

1. **Notes References**: Stored under `refs/notes/*` (e.g., `refs/notes/commits`, `refs/notes/claude-conversations`)
2. **Tree Structure**: Each notes ref points to a tree where filenames are commit SHAs
3. **Blob Content**: The actual note content is stored as blobs

The scripts use the following GitHub API endpoints:
- `/repos/{owner}/{repo}/git/refs` - List all references
- `/repos/{owner}/{repo}/git/refs/notes/{ref}` - Get specific notes reference
- `/repos/{owner}/{repo}/git/commits/{sha}` - Get commit object
- `/repos/{owner}/{repo}/git/trees/{sha}` - Get tree listing
- `/repos/{owner}/{repo}/git/blobs/{sha}` - Get blob content

## Features

- **Short SHA Support**: Scripts handle both abbreviated and full commit SHAs
- **JSON Formatting**: Automatically pretty-prints JSON note content
- **Error Handling**: Clear error messages for missing notes or repos
- **Multiple Refs**: Can work with any notes reference, not just the default

## Common Issues

1. **No notes found**: Make sure notes are pushed with `git push origin refs/notes/*`
2. **Authentication**: Ensure `gh` is authenticated with `gh auth login`
3. **Repository access**: Verify you have read access to the repository

## Future Enhancements

These scripts provide a foundation for building more advanced git notes tooling:
- Sync notes between local and remote repositories
- Search notes content across commits
- Export notes to different formats
- Integrate with CI/CD pipelines
- Create web interfaces for viewing notes