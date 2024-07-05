#!/bin/bash

echo "Installing code-server"

password={{ .Password }}
port={{ .Port }}

curl -fsSL https://code-server.dev/install.sh | sh

# Ensure the configuration directory exists
mkdir -p /root/.config/code-server

# Create or overwrite the config file with new settings
cat > /root/.config/code-server/config.yaml <<EOF
bind-addr: 0.0.0.0:$port
auth: password
password: $password
cert: false
EOF

code-server