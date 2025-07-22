#!/bin/bash
# Pre-push hook to push git notes along with commits

# This hook is called with the following parameters:
# $1 -- Name of the remote to which the push is being done
# $2 -- URL to which the push is being done

remote="$1"
url="$2"

# Prevent recursive calls when pushing notes
if [ "$GIT_NOTES_PUSH_IN_PROGRESS" = "1" ]; then
    exit 0
fi

# Check if we have any notes to push
if git show-ref --quiet refs/notes/claude-conversations; then
    echo "Pushing git notes to $remote..."
    # Set flag to prevent recursion
    export GIT_NOTES_PUSH_IN_PROGRESS=1
    git push "$remote" refs/notes/claude-conversations
    unset GIT_NOTES_PUSH_IN_PROGRESS
    
    if [ $? -eq 0 ]; then
        echo "✓ Git notes pushed successfully"
    else
        echo "⚠️  Failed to push git notes"
        echo "Push will continue, but you may need to manually push notes with:"
        echo "    git push $remote refs/notes/claude-conversations"
    fi
else
    echo "No git notes found to push"
fi

# Exit with 0 to allow the push to proceed
exit 0
