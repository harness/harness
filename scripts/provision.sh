#!/bin/sh
set -e

# Ensure that agent forwarding is set up properly.
if [ "$SSH_AUTH_SOCK" ]; then
    echo "SSH_AUTH_SOCK is set; you are successfully agent-forwarding. These keys are loaded:"
    if [ -z "$1" ]; then
        ssh-add -l
    else
        echo "Not attempting to list keys because windows ssh-agent communication is broken..."
    fi
else
    echo "No SSH_AUTH_SOCK was found in the environment!"
    exit 3
fi


# System packages
echo "Installing Base Packages"
export DEBIAN_FRONTEND=noninteractive
sudo apt-get update -qq
sudo apt-get install -qqy --force-yes build-essential bzr git mercurial vim


# Install Go
GOVERSION="1.2"
GOTARBALL="go${GOVERSION}.linux-amd64.tar.gz"
export GOROOT=/usr/local/go
export GOPATH=/opt/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin

echo "Installing Go $GOVERSION"
if [ ! $(which go) ]; then
    echo "    Downloading $GOTARBALL"
    wget --quiet --directory-prefix=/tmp https://go.googlecode.com/files/$GOTARBALL

    echo "    Extracting $GOTARBALL to $GOROOT"
    sudo tar -C /usr/local -xzf /tmp/$GOTARBALL

    echo "    Configuring GOPATH"
    sudo mkdir -p $GOPATH/src $GOPATH/bin $GOPATH/pkg
    sudo chown -R vagrant $GOPATH

    echo "    Configuring env vars"
    echo "export PATH=\$PATH:$GOROOT/bin:$GOPATH/bin" | sudo tee /etc/profile.d/golang.sh > /dev/null
    echo "export GOROOT=$GOROOT" | sudo tee --append /etc/profile.d/golang.sh > /dev/null
    echo "export GOPATH=$GOPATH" | sudo tee --append /etc/profile.d/golang.sh > /dev/null
fi


# Install drone
echo "Building Drone"
cd $GOPATH/src/github.com/drone/drone
make deps
make embed
make build
make dpkg

echo "Installing Drone"
sudo dpkg -i deb/drone.deb


# Cleanup
sudo apt-get autoremove
