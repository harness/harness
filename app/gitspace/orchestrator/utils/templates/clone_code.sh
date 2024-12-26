#!/bin/sh

repo_url="{{ .RepoURL }}"
image="{{ .Image }}"
branch="{{ .Branch }}"
repo_name="{{ .RepoName }}"
name="{{ .Name }}"
email="{{ .Email }}"

# Function to print the latest commit SHA from the remote repository
print_latest_commit() {
    echo "Fetching latest commit SHA from remote repository..."

    # Get the latest commit SHA from the remote branch
    latest_commit_sha=$(git ls-remote "$repo_url" "$branch" | awk '{print $1}')

    if [ -n "$latest_commit_sha" ]; then
        echo "Latest commit SHA: $latest_commit_sha"
    else
        echo "Failed to fetch the latest commit SHA."
    fi
    echo ""
}

# Function to print the top 10 commits after cloning
print_top_commits() {
    echo "Fetching top 10 commits from the cloned repository..."
    git log -n 10 --pretty=format:"%h %s" | while IFS= read -r commit; do
        echo "Commit SHA: $commit"
    done
    echo ""
}

# Print the latest commit before cloning
print_latest_commit

# Clone the repository inside the working directory if it doesn't exist
if [ ! -d "$HOME/$repo_name/.git" ]; then
    echo "Cloning the repository..."
    if ! git clone "$repo_url" --branch "$branch" "$HOME/$repo_name" 2>&1; then
      echo "Failed to clone the repository. Exiting..." >&2
      exit 1
    fi
else
    echo "Repository already exists. Skipping clone."
fi

# Navigate to the repository directory after cloning
cd "$HOME/$repo_name" || exit 0

# Print top 10 commits from the cloned repository
print_top_commits

# Configure Git directory
git config --global --add safe.directory "$HOME/$repo_name"

# Check if .devcontainer/devcontainer.json exists
if [ ! -f "$HOME/$repo_name/.devcontainer/devcontainer.json" ]; then
    echo "Creating .devcontainer directory and devcontainer.json..."
    mkdir -p "$HOME/$repo_name/.devcontainer"
    cat <<EOL > "$HOME/$repo_name/.devcontainer/devcontainer.json"
{
    "image": "$image"
}
EOL
    echo "devcontainer.json created."
else
    echo ".devcontainer/devcontainer.json already exists. Skipping creation."
fi

# Set user name and email if provided
if [ -n "$name" ]; then
    git config --global user.name "$name"
else
    echo "No user name configured."
fi

if [ -n "$email" ]; then
    git config --global user.email "$email"
else
    echo "No user email configured."
fi
