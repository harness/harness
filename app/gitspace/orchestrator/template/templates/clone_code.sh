#!/bin/sh

repo_url="{{ .RepoURL }}"
image="{{ .Image }}"
branch="{{ .Branch }}"
repo_name="{{ .RepoName }}"
name="{{ .Name }}"
email="{{ .Email }}"

# Clone the repository inside the working directory if it doesn't exist
if [ ! -d "$HOME/$repo_name/.git" ]; then
    echo "Cloning the repository..."
    if ! git clone "$repo_url" --branch "$branch" "$HOME/$repo_name"; then
      echo "Failed to clone the repository. Exiting..."
      exit 1
    fi
else
    echo "Repository already exists. Skipping clone."
fi

git config --global --add safe.directory "$HOME/$repo_name"

# Check if .devcontainer/devcontainer.json exists
if [ ! -f "$HOME/$repo_name/.devcontainer/devcontainer.json" ]; then
    echo "Creating .devcontainer directory and devcontainer.json..."
    mkdir -p $HOME/$repo_name/.devcontainer
    cat <<EOL > $HOME/$repo_name/.devcontainer/devcontainer.json
{
    "image": "$image"
}
EOL
    echo "devcontainer.json created."
else
    echo ".devcontainer/devcontainer.json already exists. Skipping creation."
fi

if [ -z "$name" ]; then
    echo "no user name configured"
else
    git config --global user.name "$name"
fi

if [ -z "$email" ]; then
    echo "no user email configured"
else
    git config --global user.email "$email"
fi
