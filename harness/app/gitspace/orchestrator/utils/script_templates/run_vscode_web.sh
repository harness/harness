#!/bin/sh

echo "Running VSCode Web"

# Default port comes from the Go templating variable {{ .Port }}
port={{ .Port }}
proxyuri="{{ .ProxyURI }}"

# Ensure the configuration directory exists
config_dir="$HOME/.config/code-server"
mkdir -p "$config_dir"

# Create or overwrite the config file with new settings
cat > "$config_dir/config.yaml" <<EOF
bind-addr: 0.0.0.0:$port
auth: none
cert: false
EOF

# Export the Proxy URI only if set
if [ -n "$proxyuri" ]; then
  export VSCODE_PROXY_URI="$proxyuri"
  echo "Exported VSCODE_PROXY_URI: $proxyuri"
fi

# Start code-server in the background
nohup code-server --disable-workspace-trust > "$HOME/code-server.log" 2>&1 &
code_server_pid=$!

# Wait for the code-server IPC socket file to exist
echo "Waiting for vscode web to start..."
while true; do
  # Check if the process is still running
  if ! kill -0 "$code_server_pid" 2>/dev/null; then
    echo "Error: code-server process has stopped unexpectedly."
    exit 1
  fi

  # Check for the IPC socket
  ipc_socket=$(find "$HOME/.local/" -type s -name "code-server-ipc.sock" 2>/dev/null)
  if [ -n "$ipc_socket" ]; then
    echo "vscode web is now running and ready."
    break
  fi
  sleep 3
done

exit 0