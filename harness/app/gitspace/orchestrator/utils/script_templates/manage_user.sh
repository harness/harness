#!/bin/sh

username="{{ .Username }}"
accessKey="{{ .AccessKey }}"
homeDir="{{ .HomeDir }}"
accessType={{ .AccessType }}

# Check if the user's home directory exists
if [ ! -d "$homeDir" ]; then
  echo "Directory $homeDir does not exist. Creating it..."
  mkdir -p "$homeDir"
  if [ $? -ne 0 ]; then
    echo "Failed to create directory $homeDir."
    exit 1
  fi
else
  echo "Directory $homeDir already exists."
fi

# Ensure the user has ownership and permissions to the home directory
currentOwner=$(stat -c '%U' "$homeDir")
if [ "$currentOwner" != "$username" ]; then
  echo "Updating ownership of $homeDir to $username..."
  chown -R "$username:$username" "$homeDir"
  if [ $? -ne 0 ]; then
    echo "Failed to update ownership of $homeDir."
    exit 1
  fi
fi

echo "Ensuring proper permissions for $homeDir..."
chmod 755 "$homeDir"
if [ $? -ne 0 ]; then
  echo "Failed to set permissions for $homeDir."
  exit 1
fi

echo "Directory setup for $username is complete."

if [ "ssh_key" = "$accessType" ] ; then
    echo "Add ssh key in $homeDir/.ssh/authorized_keys"
    mkdir -p $homeDir/.ssh
    chmod 700 $homeDir/.ssh
    echo $accessKey > $homeDir/.ssh/authorized_keys
    chmod 600 $homeDir/.ssh/authorized_keys
    chown -R $username:$username $homeDir/.ssh
    echo "$username:" | chpasswd -e
elif [ "user_credentials" = "$accessType"  ] ; then
    echo "$username:$accessKey" | chpasswd
else
  echo "Unsupported accessType $accessType" >&2
  exit 1
fi