#!/bin/bash

repo_url={{ .RepoURL }}
devcontainer_present={{ .DevcontainerPresent }}
image={{ .Image }}
branch={{ .Branch }}

# Extract the repository name from the URL
repo_name=$(basename -s .git "$repo_url")

# Check if Git is installed
if ! command -v git &>/dev/null; then
    echo "Git is not installed. Installing Git..."
    apt-get update
    apt-get install -y git
fi

if ! command -v git &>/dev/null; then
    echo "Git is not installed. Exiting..."
    exit 1
fi

# Clone the repository only if it doesn't exist
if [ ! -d "$repo_name" ]; then
    echo "Cloning the repository..."
    git clone "$repo_url" --branch "$branch"
else
    echo "Repository already exists. Skipping clone."
fi

# Check if devcontainer_present is set to false
if [ "$devcontainer_present" = "false" ]; then
    # Ensure the repository is cloned
    if [ -d "$repo_name" ]; then
        echo "Creating .devcontainer directory and devcontainer.json..."
        mkdir -p "$repo_name/.devcontainer"
        cat <<EOL > "$repo_name/.devcontainer/devcontainer.json"
{
    "image": "$image"
}
EOL
        echo "devcontainer.json created."
    else
        echo "Repository directory not found. Cannot create .devcontainer."
    fi
else
    echo "devcontainer_present is set to true. Skipping .devcontainer creation."
fi

echo "Script completed."
