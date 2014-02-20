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

# apt-get update.
#if [ -e /root/package-list-updated ]; then
#    echo "Skipping package cache update. To force, remove /root/package-list-updated and re-provision."
#else
#    echo "Updating package cache."
#    sudo apt-get update -qq
#	touch /root/package-list-updated
#fi

# FIXME: Don't run this every time?
sudo apt-get update -qq

echo "Installing Base Packages"
export DEBIAN_FRONTEND=noninteractive
( sed -e 's/#.*$//' | xargs sudo apt-get install -qqy --force-yes ) <<-EOF
	build-essential

    # These are needed for go get
    bzr
	git
    mercurial

    # Other
    vim

	# Stuff required by medley
	#python-software-properties	# TODO why do we need this?
	#curl						# many scripts expect this to fetch urls.
	#python-dev					# for compiling python modules
	#python-setuptools			# for installing/making packages
	#python-unittest2			# standard unit testing library
	#python-virtualenv			# for partioning python projects

	#python-lxml					# TODO why do we need this?
	#libxml2						# TODO why do we need this?
	#libxml2-dev					# TODO why do we need this?
	#libxslt1-dev				# TODO why do we need this?
EOF


# Install Go
go_version="1.2"
go_tarball="go${go_version}.linux-amd64.tar.gz"
go_root=/usr/local/go
go_path=/opt/go

echo "Installing Go $go_version"
if [ ! $(which go) ]; then
    echo "    Downloading $go_tarball"
    wget --quiet --directory-prefix=/tmp https://go.googlecode.com/files/$go_tarball

    echo "    Extracting $go_tarball to $go_root"
    sudo tar -C /usr/local -xzf /tmp/$go_tarball

    echo "    Configuring GOPATH"
    sudo mkdir -p $go_path/src $go_path/bin $go_path/pkg
    sudo chown -R vagrant $go_path

    echo "    Configuring env vars"
    echo "export PATH=\$PATH:$go_root/bin" | sudo tee /etc/profile.d/golang.sh > /dev/null
    echo "export GOROOT=$go_root" | sudo tee --append /etc/profile.d/golang.sh > /dev/null
    echo "export GOPATH=$go_path" | sudo tee --append /etc/profile.d/golang.sh > /dev/null
fi


# Cleanup
sudo apt-get autoremove
