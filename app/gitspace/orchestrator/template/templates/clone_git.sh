#!/bin/sh

repo_url={{ .RepoURL }}
image={{ .Image }}
branch={{ .Branch }}
password={{ .Password }}
email={{ .Email }}
name={{ .Name }}

# Extract the repository name from the URL
repo_name=$(basename -s .git "$repo_url")

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
git config --global --add safe.directory /$repo_name
git config --global user.email $email
git config --global user.name $name
# Clone the repository inside the working directory if it doesn't exist
if [ ! -d ".git" ]; then
    echo "Cloning the repository..."
git clone "$repo_url" --branch "$branch" /$repo_name
else
    echo "Repository already exists. Skipping clone."
fi
rm $HOME/.git-askpass
# Check if .devcontainer/devcontainer.json exists
if [ ! -f ".devcontainer/devcontainer.json" ]; then
    echo "Creating .devcontainer directory and devcontainer.json..."
    mkdir -p /$repo_name/.devcontainer
    cat <<EOL > /$repo_name/.devcontainer/devcontainer.json
{
    "image": "$image"
}
EOL
    echo "devcontainer.json created."
else
    echo ".devcontainer/devcontainer.json already exists. Skipping creation."
fi
