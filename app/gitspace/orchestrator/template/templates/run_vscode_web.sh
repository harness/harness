#!/bin/sh

echo "Running VSCode Web"

port={{ .Port }}

# Ensure the configuration directory exists
mkdir -p $HOME/.config/code-server

# Create or overwrite the config file with new settings
cat > $HOME/.config/code-server/config.yaml <<EOF
bind-addr: 0.0.0.0:$port
auth: none
cert: false
EOF

code-server --disable-workspace-trust