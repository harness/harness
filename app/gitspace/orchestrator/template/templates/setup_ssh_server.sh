#!/bin/sh

osInfoScript={{ .OSInfoScript }}

eval "$osInfoScript"

# Install SSH if it's not already installed
if ! command -v sshd >/dev/null 2>&1; then
    echo "OpenSSH server is not installed. Installing..."

    case "$(distro)" in
        debian)
            apt-get update
            apt-get install -y openssh-server
          ;;
        fedora)
            dnf install -y openssh-server
          ;;
        opensuse)
            zypper install -y openssh
          ;;
        alpine)
          apk add openssh
          ;;
        arch)
          pacman -Syu --noconfirm openssh
          ;;
        freebsd)
          pkg install -y openssh-portable
          ;;
        *)
          echo "Unsupported distribution: $distro."
          exit 1
          ;;
      esac


else
    echo "OpenSSH server is already installed."
fi

username={{ .Username }}
accessType={{ .AccessType }}
repoName="{{ .RepoName }}"

# Configure SSH to allow this user
config_file='/etc/ssh/sshd_config'

grep -q "^AllowUsers" $config_file
if [ $? -eq 0 ]; then
   # If AllowUsers exists, overwrite all existing users with new user
    sed -i "s/^AllowUsers.*/AllowUsers $username/" $config_file
else
    # Otherwise, add a new AllowUsers line
    echo "AllowUsers $username" >> $config_file
fi

echo "Access type $accessType"

if [ "ssh_key" = "$accessType" ] ; then
# Ensure password authentication is disabled
sed -i 's/^PasswordAuthentication yes/PasswordAuthentication no/' $config_file
if ! grep -q "^PasswordAuthentication no" $config_file; then
    echo "PasswordAuthentication no" >> $config_file
fi
sed -i 's/^UsePAM yes/UsePAM no/' $config_file
echo "AuthorizedKeysFile	.ssh/authorized_keys" >> $config_file
echo "PubkeyAuthentication yes" >> $config_file
else
# Ensure password authentication is enabled
if ! grep -q "^PasswordAuthentication yes" $config_file; then
    echo "PasswordAuthentication yes" >> $config_file
fi
if ! grep -q "^PermitEmptyPasswords yes" $config_file; then
    echo "PermitEmptyPasswords yes" >> $config_file
fi
if ! grep -q "^PermitRootLogin yes" $config_file; then
    echo "PermitRootLogin yes" >> $config_file
fi
fi

mkdir -p /var/run/sshd

# Create .vscode/extensions.json for the current user
USER_HOME=$(eval echo ~$username)
VSCODE_REMOTE_DIR="$USER_HOME/$repoName/.vscode"
EXTENSIONS_FILE="$VSCODE_REMOTE_DIR/extensions.json"

# Create .vscode directory with correct ownership
if [ ! -d "$VSCODE_REMOTE_DIR" ]; then
    echo "Creating directory: $VSCODE_REMOTE_DIR"
    mkdir -p "$VSCODE_REMOTE_DIR"
    chown "$username:$username" "$VSCODE_REMOTE_DIR"
fi

# Create extensions.json file with correct ownership
if [ ! -f "$EXTENSIONS_FILE" ]; then
    echo "Creating extensions.json file for user $username"
    cat <<EOF > "$EXTENSIONS_FILE"
{
    "recommendations": {{ .Extensions }}
}
EOF
    chmod 644 "$EXTENSIONS_FILE"
    chown "$username:$username" "$EXTENSIONS_FILE"
else
    echo "extensions.json already exists for user $username"
fi
