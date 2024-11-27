#!/bin/sh

echo "Running VSCode Web"

# Default port comes from the Go templating variable {{ .Port }}
port={{ .Port }}
proxyuri="{{ .ProxyURI }}"
extensions={{ range .Extensions }}"{{ . }}" {{ end }}

# Ensure the configuration directory exists
mkdir -p $HOME/.config/code-server

# Create or overwrite the config file with new settings
cat > $HOME/.config/code-server/config.yaml <<EOF
bind-addr: 0.0.0.0:$port
auth: none
cert: false
EOF

# Install extensions using code-server CLI and display errors if any
for extension in $extensions; do
  echo "Installing extension: $extension"
  if ! code-server --install-extension "$extension"; then
    echo "Error installing extension: $extension" >&2
  fi
done

# Export the Proxy URI only if set
if [ -n "$proxyuri" ]; then
  export VSCODE_PROXY_URI="$proxyuri"
  echo "Exported VSCODE_PROXY_URI: $proxyuri"
fi

# Run code-server with templated arguments
eval "code-server --disable-workspace-trust"