#!/bin/sh
extensions={{ range .Extensions }}"{{ . }}" {{ end }}

echo "Installing VSCode Web"

curl -fsSL https://code-server.dev/install.sh | sh

# Install extensions using code-server CLI and display errors if any
for extension in $extensions; do
  echo "Installing extension: $extension"
  if ! code-server --install-extension "$extension"; then
    echo "Error installing extension: $extension" >&2
  fi
done
