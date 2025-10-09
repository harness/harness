#!/bin/sh

SSH_PORT={{ .Port }}

config_file='/etc/ssh/sshd_config'

HOST_KEYS="/etc/ssh/ssh_host_rsa_key /etc/ssh/ssh_host_ecdsa_key /etc/ssh/ssh_host_ed25519_key"
for KEY in $HOST_KEYS; do
    if [ ! -f "$KEY" ]; then
        echo "Generating host key: $KEY"
        case "$KEY" in
            "/etc/ssh/ssh_host_rsa_key")
                ssh-keygen -t rsa -b 4096 -f "$KEY" -N ""
                ;;
            "/etc/ssh/ssh_host_ecdsa_key")
                ssh-keygen -t ecdsa -b 521 -f "$KEY" -N ""
                ;;
            "/etc/ssh/ssh_host_ed25519_key")
                ssh-keygen -t ed25519 -f "$KEY" -N ""
                ;;
        esac
        chmod 600 "$KEY"
        chmod 644 "${KEY}.pub"
    else
        echo "Host key already exists: $KEY"
    fi
done

# Change the default SSH port
sed -i "s/^#Port 22/Port $SSH_PORT/" $config_file
if ! grep -q "^Port $SSH_PORT" $config_file; then
    echo "Port $SSH_PORT" >> $config_file
fi

echo "Running SSH Server on port $SSH_PORT"

/usr/sbin/sshd