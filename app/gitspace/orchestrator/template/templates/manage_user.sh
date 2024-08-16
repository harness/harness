#!/bin/sh

username={{ .Username }}
accessKey={{ .AccessKey }}
homeDir={{ .HomeDir }}
accessType={{ .AccessType }}

# Check if the user already exists
if id "$username" >/dev/null 2>&1; then
    echo "User $username already exists."
else
    # Create a new user
    adduser --disabled-password --home "$homeDir" --gecos "" "$username"
    if [ $? -ne 0 ]; then
        echo "Failed to create user $username."
        exit 1
    fi
fi

# Changing ownership of everything inside user home to the newly created user
chown -R $username:$username $homeDir
echo "Changing ownership of dir $homeDir to $username."

if $accessType = "ssh_key"; then
    echo $accessKey > $homeDir/.ssh/authorized_keys
else
    echo "$username:$accessKey" | chpasswd
fi