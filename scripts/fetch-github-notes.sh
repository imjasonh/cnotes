#!/bin/bash

# Script to fetch git notes from GitHub using gh CLI
# Usage: ./fetch-github-notes.sh [owner/repo] [commit-sha] [notes-ref]

set -e

# Default to current repo if not specified
if [ -z "$1" ]; then
    REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)
else
    REPO="$1"
fi

COMMIT_SHA="$2"
NOTES_REF_NAME="${3:-claude-conversations}"  # Default to claude-conversations

echo "Fetching git notes from $REPO (ref: refs/notes/$NOTES_REF_NAME)..."

# Step 1: Get the notes reference
echo "Step 1: Getting notes reference..."
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
echo "  Notes ref SHA: $NOTES_SHA"

# Step 2: Get the commit object to find the tree
echo "Step 2: Getting commit object..."
COMMIT=$(gh api "repos/$REPO/git/commits/$NOTES_SHA")
TREE_SHA=$(echo "$COMMIT" | jq -r '.tree.sha')
echo "  Tree SHA: $TREE_SHA"

# Step 3: Get the tree to list all noted commits
echo "Step 3: Getting tree (list of noted commits)..."
TREE=$(gh api "repos/$REPO/git/trees/$TREE_SHA")

# If a specific commit was requested, look for it
if [ -n "$COMMIT_SHA" ]; then
    echo "Looking for notes on commit $COMMIT_SHA..."
    
    # Find the blob for this specific commit (handle both short and full SHAs)
    BLOB_SHA=$(echo "$TREE" | jq -r ".tree[] | select(.path | startswith(\"$COMMIT_SHA\")) | .sha" | head -1)
    
    if [ -z "$BLOB_SHA" ] || [ "$BLOB_SHA" = "null" ]; then
        echo "No notes found for commit $COMMIT_SHA"
        exit 1
    fi
    
    # Step 4: Get the blob content
    echo "Step 4: Getting note content..."
    BLOB=$(gh api "repos/$REPO/git/blobs/$BLOB_SHA")
    CONTENT=$(echo "$BLOB" | jq -r '.content' | base64 -d)
    
    echo ""
    echo "=== Note for commit $COMMIT_SHA ==="
    echo "$CONTENT"
else
    # List all commits with notes
    echo ""
    echo "=== Commits with notes ==="
    echo "$TREE" | jq -r '.tree[] | .path' | while read -r commit_sha; do
        echo "  $commit_sha"
    done
    
    echo ""
    echo "To see a specific note, run:"
    echo "  $0 $REPO <commit-sha>"
fi