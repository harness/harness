#!/bin/sh

repo_url={{ .RepoURL }}
image={{ .Image }}
branch={{ .Branch }}
repo_name={{ .RepoName }}

password={{ .Password }}
email={{ .Email }}
name={{ .Name }}
host={{ .Host }}
protocol={{ .Protocol }}
path={{ .Path }}

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
if [ -z "$password" ]; then
    echo "setting up without credentials"
else
    git config --global credential.helper 'cache --timeout=2592000'
    git config --global user.email "$email"
    git config --global user.name "$name"
    touch .gitcontext
    echo "host="$host >> .gitcontext
    echo "protocol="$protocol >> .gitcontext
    echo "path="$path >> .gitcontext
    echo "username="$email >> .gitcontext
    echo "password="$password >> .gitcontext
    echo "" >> .gitcontext

    cat .gitcontext | git credential approve
    rm .gitcontext
fi

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
