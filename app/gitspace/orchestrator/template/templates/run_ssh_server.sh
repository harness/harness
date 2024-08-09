#!/bin/sh

SSH_PORT={{ .Port }}

config_file='/etc/ssh/sshd_config'

# Change the default SSH port
sed -i "s/^#Port 22/Port $SSH_PORT/" $config_file
if ! grep -q "^Port $SSH_PORT" $config_file; then
    echo "Port $SSH_PORT" >> $config_file
fi

echo "Running SSH Server on port $SSH_PORT"

/usr/sbin/sshd