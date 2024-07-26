#!/bin/sh

echo "Installing VSCode Web"

curl -fsSL https://code-server.dev/install.sh | sh

port={{ .Port }}

# Ensure the configuration directory exists
mkdir -p /root/.config/code-server

# Create or overwrite the config file with new settings
cat > /root/.config/code-server/config.yaml <<EOF
bind-addr: 0.0.0.0:$port
auth: none
cert: false
EOF