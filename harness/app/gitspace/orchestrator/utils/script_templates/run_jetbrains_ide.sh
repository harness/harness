#!/bin/sh

username={{ .Username }}
repoName="{{ .RepoName }}"
ideDirName="{{ .IdeDirName}}"

# Create .cache/JetBrains for the current user
USER_HOME=$(eval echo ~$username)
INTELLIJ_PATH="$USER_HOME/$ideDirName"

# check is intellij is downloaded
if [ ! -d "$INTELLIJ_PATH" ]; then
  echo "IntelliJ IDE is not downloaded."
  exit 1
fi

echo "registering remote Intellij IDE..."
INTELLIJ_LOG_FILE="$INTELLIJ_PATH/jetbrains.log"
echo "storing ide logs in $INTELLIJ_LOG_FILE..."
nohup "$INTELLIJ_PATH/bin/remote-dev-server.sh" run "$USER_HOME/$repoName" --ssh-link-user "$username" > "$INTELLIJ_LOG_FILE" 2>&1 &

is_ide_running(){
  retries=5
  sleep_interval=2
  log_entry="Gateway link: "

  # Loop for the retry count
  iteration_cnt=1
  while [ $iteration_cnt -le $retries ]; do
    echo "checking ide logs, iteration: #$iteration_cnt... "
    # Check for gateway link log to confirm everything is working
    if grep -q "$log_entry" "$INTELLIJ_LOG_FILE"; then
        # If the entry is found, return true
        return 0
    fi
    # Wait for the interval before retrying
    sleep $sleep_interval
    iteration_cnt=$((iteration_cnt + 1))
  done
}

echo "waiting for ide to run..."
if is_ide_running; then
  echo "ide is running"
else
  echo "ide is not running"
  exit 1
fi