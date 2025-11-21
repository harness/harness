#!/bin/sh

username={{ .Username }}
ideDirName="{{ .IdeDirName}}"
idePlugins="{{ .IdePluginsName }}"

USER_HOME=$(eval echo ~$username)
INTELLIJ_PATH="$USER_HOME/$ideDirName"

# check is intellij is downloaded
if [ ! -d "$INTELLIJ_PATH" ]; then
  echo "IntelliJ IDE is not downloaded."
  exit 1
fi

echo "installing Jetbrains plugins: $idePlugins"
JETBRAINS_PLUGIN_INSTALL_LOGS="$INTELLIJ_PATH/jetbrains_install.log"
eval "$INTELLIJ_PATH/bin/remote-dev-server.sh installPlugins $idePlugins > $JETBRAINS_PLUGIN_INSTALL_LOGS 2>&1"

# check plugin status
check_plugin_status() {
  plugin="$1"

  if grep -q "already installed: $plugin" "$JETBRAINS_PLUGIN_INSTALL_LOGS"; then
    echo "$plugin already installed."
  elif grep -q "installed plugin: PluginNode{id=$plugin" "$JETBRAINS_PLUGIN_INSTALL_LOGS"; then
    echo "$plugin installed successfully."
  else
    echo "$plugin installation failed."
  fi
}

for plugin in $idePlugins; do
  check_plugin_status "$plugin"
done
