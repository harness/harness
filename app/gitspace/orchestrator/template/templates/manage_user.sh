#!/bin/sh

username={{ .Username }}
password={{ .Password }}
workingDir={{ .WorkingDirectory }}

# Check if the user already exists
if id "$username" >/dev/null 2>&1; then
    echo "User $username already exists."
else
    # Create a new user
    adduser --disabled-password --gecos "" "$username"
    if [ $? -ne 0 ]; then
        echo "Failed to create user $username."
        exit 1
    fi
fi

# Set or update the user's password using chpasswd
echo "$username:$password" | chpasswd

# Changing ownership of everything inside user home to the newly created user
chown -R $username $workingDir
echo "Changing ownership of dir $workingDir to user $username."