#!/bin/bash

# Create symlinks to all git hooks in your own .git dir.
echo "### Creating symlinks to our git hooks..."
for f in $(ls -d githooks/*)
do ln -s ../../$f .git/hooks
done && ls -al .git/hooks | grep githooks
echo ""
