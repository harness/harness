#!/bin/sh

osInfoScript={{ .OSInfoScript }}

eval "$osInfoScript"

# Detect OS and architecture
OS=$(uname -s | tr "[:upper:]" "[:lower:]")
ARCH=$(uname -m | sed "s/x86_64/amd64/;s/aarch64/arm64/")

# Create .bin directory if not exists
BIN_DIR="$HOME/.bin"
mkdir -p "$BIN_DIR"

# Download agentapi binary
AGENTAPI_URL="https://github.com/coder/agentapi/releases/latest/download/agentapi-${OS}-${ARCH}"
AGENTAPI_PATH="$BIN_DIR/agentapi"

echo "Downloading agentapi from: $AGENTAPI_URL"
curl -fsSL "$AGENTAPI_URL" -o "$AGENTAPI_PATH"
chmod +x "$AGENTAPI_PATH"

echo "Downloaded agentapi to $AGENTAPI_PATH"

# Run agentapi in background with logs
LOG_FILE="$BIN_DIR/agentapi.log"
nohup "$AGENTAPI_PATH" server --allowed-hosts '*' -- claude --dangerously-skip-permissions > "$LOG_FILE" 2>&1 &

echo "agentapi is running in background. Logs: $LOG_FILE"