#!/bin/sh

username={{ .Username }}
repoName="{{ .RepoName }}"

# Create .vscode/extensions.json for the current user
USER_HOME=$(eval echo ~$username)
VSCODE_REMOTE_DIR="$USER_HOME/$repoName/.vscode"
EXTENSIONS_FILE="$VSCODE_REMOTE_DIR/extensions.json"

# Create .vscode directory with correct ownership
if [ ! -d "$VSCODE_REMOTE_DIR" ]; then
    echo "Creating directory: $VSCODE_REMOTE_DIR"
    mkdir -p "$VSCODE_REMOTE_DIR"
    chown "$username:$username" "$VSCODE_REMOTE_DIR"
fi

# Create extensions.json file with correct ownership
if [ ! -f "$EXTENSIONS_FILE" ]; then
    echo "Creating extensions.json file for user $username"
    cat <<EOF > "$EXTENSIONS_FILE"
{
    "recommendations": {{ .Extensions }}
}
EOF
    chmod 644 "$EXTENSIONS_FILE"
    chown "$username:$username" "$EXTENSIONS_FILE"
else
    echo "extensions.json already exists for user $username"
fi
