#!/bin/sh

password={{ .Password }}

# Create or overwrite the config file with new settings
touch $HOME/.git-askpass
cat > $HOME/.git-askpass <<EOF
echo $password
EOF
chmod 700 $HOME/.git-askpass
git config --global credential.helper 'cache --timeout=2592000'
#run git operation to cache the credential in memory
export GIT_ASKPASS=$HOME/.git-askpass
git ls-remote
rm $HOME/.git-askpass