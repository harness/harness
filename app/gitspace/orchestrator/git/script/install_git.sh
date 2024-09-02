#!/bin/sh

# Check if Git is installed
if ! command -v git >/dev/null 2>&1; then
    echo "Git is not installed. Installing Git..."
    apt-get update
    apt-get install -y git
fi

if ! command -v git >/dev/null 2>&1; then
    echo "Git is not installed. Exiting..."
    exit 1
fi