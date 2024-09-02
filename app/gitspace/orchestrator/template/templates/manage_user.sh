#!/bin/sh

username={{ .Username }}
accessKey="{{ .AccessKey }}"
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
chmod 755 $homeDir

if [ "ssh_key" = "$accessType" ] ; then
    echo "Add ssh key in $homeDir/.ssh/authorized_keys"
    mkdir -p $homeDir/.ssh
    chmod 700 $homeDir/.ssh
    echo $accessKey > $homeDir/.ssh/authorized_keys
    chmod 600 $homeDir/.ssh/authorized_keys
    chown -R $username:$username $homeDir/.ssh
    echo "$username:$username" | chpasswd
elif [ "user_credentials" = "$accessType"  ] ; then
    echo "$username:$accessKey" | chpasswd
else
  echo "Unsupported accessType $accessType" >&2
  exit 1
fi