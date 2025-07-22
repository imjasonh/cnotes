#!/bin/bash

# Script to fetch ALL git notes from GitHub and display them nicely
# Usage: ./fetch-all-github-notes.sh [owner/repo] [notes-ref]

set -e

# Default to current repo if not specified
if [ -z "$1" ]; then
    REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)
else
    REPO="$1"
fi

NOTES_REF_NAME="${2:-claude-conversations}"  # Default to claude-conversations

echo "Fetching all git notes from $REPO (ref: refs/notes/$NOTES_REF_NAME)..."
echo ""

# Step 1: Get the notes reference
NOTES_REF=$(gh api "repos/$REPO/git/refs/notes/$NOTES_REF_NAME" 2>/dev/null || echo "")

if [ -z "$NOTES_REF" ]; then
    echo "No notes found in this repository (refs/notes/$NOTES_REF_NAME doesn't exist)"
    echo "Make sure notes have been pushed with: git push origin refs/notes/$NOTES_REF_NAME"
    exit 1
fi

NOTES_SHA=$(echo "$NOTES_REF" | jq -r '.object.sha // empty')
if [ -z "$NOTES_SHA" ]; then
    echo "Failed to get notes SHA from response"
    exit 1
fi

# Step 2: Get the tree listing all noted commits
TREE_SHA=$(gh api "repos/$REPO/git/commits/$NOTES_SHA" | jq -r '.tree.sha')
TREE=$(gh api "repos/$REPO/git/trees/$TREE_SHA")

# Count total notes
TOTAL=$(echo "$TREE" | jq '.tree | length')
echo "Found $TOTAL commits with notes"
echo ""

# Step 3: Fetch and display each note
echo "$TREE" | jq -r '.tree[] | "\(.path) \(.sha)"' | while read -r commit_sha blob_sha; do
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Commit: $commit_sha"
    
    # Get commit info (author, date, message)
    COMMIT_INFO=$(gh api "repos/$REPO/git/commits/$commit_sha" 2>/dev/null || echo "{}")
    if [ "$COMMIT_INFO" != "{}" ]; then
        AUTHOR=$(echo "$COMMIT_INFO" | jq -r '.author.name // "Unknown"')
        DATE=$(echo "$COMMIT_INFO" | jq -r '.author.date // "Unknown"')
        MESSAGE=$(echo "$COMMIT_INFO" | jq -r '.message' | head -n1)
        
        echo "Author: $AUTHOR"
        echo "Date:   $DATE"
        echo "Message: $MESSAGE"
    fi
    
    echo ""
    echo "Note content:"
    echo "────────────"
    
    # Get the note content
    BLOB=$(gh api "repos/$REPO/git/blobs/$blob_sha")
    CONTENT=$(echo "$BLOB" | jq -r '.content' | base64 -d)
    
    # Indent the note content
    echo "$CONTENT" | sed 's/^/  /'
    echo ""
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"