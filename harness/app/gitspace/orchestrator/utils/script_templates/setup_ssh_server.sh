#!/bin/sh

osInfoScript={{ .OSInfoScript }}

eval "$osInfoScript"

# Install SSH if it's not already installed
if ! command -v sshd >/dev/null 2>&1; then
    echo "OpenSSH server is not installed. Installing..."

    case "$(distro)" in
        debian)
            export DEBIAN_FRONTEND=noninteractive && apt-get update
            export DEBIAN_FRONTEND=noninteractive && apt-get install -y openssh-server
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

