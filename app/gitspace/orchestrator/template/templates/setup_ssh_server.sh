#!/bin/sh

# Install SSH if it's not already installed
if ! command -v sshd >/dev/null 2>&1; then
    echo "OpenSSH server is not installed. Installing..."
    apt-get update
    apt-get install -y openssh-server
else
    echo "OpenSSH server is already installed."
fi

username={{ .Username }}
password={{ .Password }}
workingDir={{ .WorkingDirectory }}

# Configure SSH to allow this user
config_file='/etc/ssh/sshd_config'
grep -q "^AllowUsers" $config_file
if [ $? -eq 0 ]; then
    # If AllowUsers exists, add the user to it
    sed -i "/^AllowUsers/ s/$/ $username/" $config_file
else
    # Otherwise, add a new AllowUsers line
    echo "AllowUsers $username" >> $config_file
fi

# Ensure password authentication is enabled
sed -i 's/^PasswordAuthentication no/PasswordAuthentication yes/' $config_file
if ! grep -q "^PasswordAuthentication yes" $config_file; then
    echo "PasswordAuthentication yes" >> $config_file
fi

mkdir /var/run/sshd