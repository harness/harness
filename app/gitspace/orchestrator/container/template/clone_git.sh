#!/bin/sh

repo_url={{ .RepoURL }}
image={{ .Image }}
branch={{ .Branch }}

# Extract the repository name from the URL
repo_name=$(basename -s .git "$repo_url")

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

# Clone the repository inside the working directory if it doesn't exist
if [ ! -d ".git" ]; then
    echo "Cloning the repository..."
    git clone "$repo_url" --branch "$branch" .
else
    echo "Repository already exists. Skipping clone."
fi

# Check if .devcontainer/devcontainer.json exists
if [ ! -f ".devcontainer/devcontainer.json" ]; then
    echo "Creating .devcontainer directory and devcontainer.json..."
    mkdir -p ".devcontainer"
    cat <<EOL > ".devcontainer/devcontainer.json"
{
    "image": "$image"
}
EOL
    echo "devcontainer.json created."
else
    echo ".devcontainer/devcontainer.json already exists. Skipping creation."
fi
