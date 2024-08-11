#!/bin/sh

repo_url={{ .RepoURL }}
image={{ .Image }}
branch={{ .Branch }}
repo_name={{ .RepoName }}

password={{ .Password }}
email={{ .Email }}
name={{ .Name }}

# Create or overwrite the config file with new settings
touch $HOME/.git-askpass
cat > $HOME/.git-askpass <<EOF
echo $password
EOF
chmod 700 $HOME/.git-askpass
export GIT_ASKPASS=$HOME/.git-askpass
git config --global credential.helper 'cache --timeout=2592000'

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

git config --global user.email $email
git config --global user.name $name
# Clone the repository inside the working directory if it doesn't exist
if [ ! -d ".git" ]; then
    echo "Cloning the repository..."
    if ! git clone "$repo_url" --branch "$branch" ; then
      echo "Failed to clone the repository. Exiting..."
      rm $HOME/.git-askpass
      exit 1
    fi
else
    echo "Repository already exists. Skipping clone."
fi
rm $HOME/.git-askpass

git config --global --add safe.directory $HOME/$repo_name

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
