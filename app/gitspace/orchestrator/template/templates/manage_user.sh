#!/bin/sh

username={{ .Username }}
accessKey="{{ .AccessKey }}"
homeDir={{ .HomeDir }}
accessType={{ .AccessType }}
osInfoScript={{ .OSInfoScript }}

eval "$osInfoScript"

# Check if the user already exists
if id "$username" >/dev/null 2>&1; then
  echo "User $username already exists."
else
  # Create a new user
  case "$(distro)" in
      debian)
        apt-get update && apt-get install -y adduser
        adduser --disabled-password --home "$homeDir" --gecos "" "$username"
        usermod -aG sudo "$username"
        echo "%sudo ALL=(ALL:ALL) NOPASSWD: ALL" >> /etc/sudoers
        ;;
      fedora)
        useradd -m -d "$homeDir" "$username"
        usermod -aG wheel "$username"
        echo "%wheel ALL=(ALL:ALL) NOPASSWD: ALL" >> /etc/sudoers
        ;;
      opensuse)
        useradd -m -d "$homeDir" "$username"
        passwd -l "$username"  # Locks the password to prevent login
        zypper in -y sudo
        groupadd sudo
        usermod -aG sudo "$username"
        echo "%sudo ALL=(ALL:ALL) NOPASSWD: ALL" >> /etc/sudoers
        chown -R "$username":sudo "$homeDir"
        ;;
      alpine)
        adduser -h "$homeDir" -s /bin/ash -D "$username"  # Default shell is ash for Alpine
        ;;
      arch)
        useradd -m -d "$homeDir" -s /bin/bash "$username"
        ;;
      freebsd)
        pw useradd -n "$username" -d "$homeDir" -m
        ;;
      *)
        echo "Unsupported distribution: $distro."
        exit 1
        ;;
    esac

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