#!/bin/bash

# Script to list all git notes attached to a particular commit via GitHub API
# Usage: ./list-commit-notes.sh [owner/repo] <commit-sha>

set -e

# Default to current repo if not specified
if [ -z "$2" ]; then
    if [ -z "$1" ]; then
        echo "Usage: $0 [owner/repo] <commit-sha>"
        exit 1
    fi
    REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)
    COMMIT_SHA="$1"
else
    REPO="$1"
    COMMIT_SHA="$2"
fi

echo "Checking for notes on commit $COMMIT_SHA in $REPO..."
echo ""

# First, get all refs that look like notes
echo "Step 1: Finding all notes references..."
ALL_REFS=$(gh api "repos/$REPO/git/refs" --paginate | jq -r '.[] | select(.ref | startswith("refs/notes/")) | .ref' | sort)

if [ -z "$ALL_REFS" ]; then
    echo "No notes references found in this repository"
    echo "Notes need to be pushed with: git push origin 'refs/notes/*'"
    exit 0
fi

echo "Found notes references:"
echo "$ALL_REFS" | sed 's/^/  /'
echo ""

# For each notes ref, check if it contains a note for our commit
FOUND_NOTES=false
TEMP_FILE=$(mktemp)

while read -r NOTE_REF; do
    NOTE_REF_NAME=$(echo "$NOTE_REF" | sed 's|refs/notes/||')
    
    # Get the notes tree for this ref
    NOTES_SHA=$(gh api "repos/$REPO/git/$NOTE_REF" 2>/dev/null | jq -r '.object.sha // empty')
    
    if [ -z "$NOTES_SHA" ]; then
        continue
    fi
    
    # Get the tree
    TREE_SHA=$(gh api "repos/$REPO/git/commits/$NOTES_SHA" 2>/dev/null | jq -r '.tree.sha // empty')
    
    if [ -z "$TREE_SHA" ]; then
        continue
    fi
    
    # Check if this tree contains our commit (handle short SHAs)
    BLOB_INFO=$(gh api "repos/$REPO/git/trees/$TREE_SHA" 2>/dev/null | jq -r ".tree[] | select(.path | startswith(\"$COMMIT_SHA\")) | \"\(.path) \(.sha)\"" | head -1)
    
    if [ -n "$BLOB_INFO" ]; then
        echo "true" > "$TEMP_FILE"
        FULL_SHA=$(echo "$BLOB_INFO" | cut -d' ' -f1)
        BLOB_SHA=$(echo "$BLOB_INFO" | cut -d' ' -f2)
        
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "Found note in: $NOTE_REF"
        echo "Full commit SHA: $FULL_SHA"
        echo ""
        
        # Get and display the note content
        BLOB=$(gh api "repos/$REPO/git/blobs/$BLOB_SHA")
        CONTENT=$(echo "$BLOB" | jq -r '.content' | base64 -d)
        
        # Check if it's JSON and pretty print if so
        if echo "$CONTENT" | jq . >/dev/null 2>&1; then
            echo "Note content (formatted JSON):"
            echo "────────────────────────────"
            echo "$CONTENT" | jq . | sed 's/^/  /'
        else
            echo "Note content:"
            echo "────────────────────────────"
            echo "$CONTENT" | sed 's/^/  /'
        fi
        echo ""
    fi
done <<< "$ALL_REFS"

# Check if we found any notes
if [ -f "$TEMP_FILE" ] && [ "$(cat "$TEMP_FILE")" = "true" ]; then
    FOUND_NOTES=true
fi
rm -f "$TEMP_FILE"

if [ "$FOUND_NOTES" = "false" ]; then
    echo "No notes found for commit $COMMIT_SHA"
    echo ""
    echo "Possible reasons:"
    echo "  - The commit doesn't have any notes attached"
    echo "  - The commit SHA is incorrect"
    echo "  - Notes haven't been pushed to GitHub"
fi