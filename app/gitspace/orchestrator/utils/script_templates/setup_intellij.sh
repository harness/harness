#!/bin/sh

username={{ .Username }}
ideDownloadUrlArm64="{{ .IdeDownloadURLArm64}}"
ideDownloadUrlAmd64="{{ .IdeDownloadURLAmd64}}"
ideDirName="{{ .IdeDirName}}"
TMP_DOWNLOAD_DIR="/tmp/harness/"

# is_arm checks if underlying image is arm based.
is_arm() {
  case "$(uname -a)" in
  *arm* ) true;;
  *arm64* ) true;;
  *aarch* ) true;;
  *aarch64* ) true;;
  * ) false;;
  esac
}

ideDownloadUrl="$ideDownloadUrlAmd64"
if is_arm; then
  echo "Detected arm architecture."
  ideDownloadUrl="$ideDownloadUrlArm64"
fi

# Create $TMP_DOWNLOAD_DIR directory with correct ownership
if [ ! -d "$TMP_DOWNLOAD_DIR" ]; then
    mkdir -p "$TMP_DOWNLOAD_DIR"
    chown "$username:$username" "$TMP_DOWNLOAD_DIR"
fi

USER_HOME=$(eval echo ~$username)
INTELLIJ_PATH="$USER_HOME/$ideDirName"

# Get the tarball file name from the URL
echo "Downloading IDE from $ideDownloadUrl ..."
TARBALL_NAME=$(basename "$ideDownloadUrl")
# Download JetBrains product
curl --no-progress-meter -L -o "$TMP_DOWNLOAD_DIR/$TARBALL_NAME" "$ideDownloadUrl"

# Verify the download
if [ $? -ne 0 ]; then
  echo "Download failed. Please check the product name, version, or URL."
  exit 1
fi

# Create  INTELLIJ_PATH directory with correct ownership
if [ ! -d "$INTELLIJ_PATH" ]; then
    echo "Creating directory: $INTELLIJ_PATH"
    mkdir -p "$INTELLIJ_PATH"
fi

echo "extracting $TARBALL_NAME..."
tar -xzf "$TMP_DOWNLOAD_DIR/$TARBALL_NAME" -C "$INTELLIJ_PATH"

EXTRACTED_DIRNAME=$(basename "$(find "$INTELLIJ_PATH" -maxdepth 1 -type d)")

mv "$INTELLIJ_PATH/$EXTRACTED_DIRNAME/"* "$INTELLIJ_PATH"
rm -r "$INTELLIJ_PATH/$EXTRACTED_DIRNAME"

if [ -d "$USER_HOME" ]; then
  chown -R "$username:$username" "$USER_HOME"
fi

# Cleanup
echo "Cleaning up tarball..."
rm -r "$TMP_DOWNLOAD_DIR"
