#!/bin/sh
extensions="{{- range $index, $extension := .Extensions }}{{ if $index }} {{ end }}{{ $extension }}{{- end }}"

echo "Installing VSCode Web"

curl -fsSL https://code-server.dev/install.sh | sh

# Install extensions using code-server CLI and display errors if any
for extension in $extensions; do
  if code-server --install-extension "$extension"; then
    echo "Successfully installed extension: $extension"
  else
    echo "Error installing extension: $extension" >&2
  fi
done
