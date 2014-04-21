# -*- mode: ruby -*-
# vi: set ft=ruby :

# Vagrantfile API/syntax version. Don't touch unless you know what you're doing!
VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  # Drone supports 12.04 64bit and 13.04 64bit
  config.vm.box = "precise64"
  config.vm.box_url = "http://files.vagrantup.com/precise64.box"

  # Forward keys from SSH agent rather than copypasta
  config.ssh.forward_agent = true

  # FIXME: Maybe this is enough
  config.vm.provider "virtualbox" do |v|
      v.customize ["modifyvm", :id, "--memory", "2048"]
  end

  # Drone by default runs on port 80. Forward from host to guest
  config.vm.network :forwarded_port, guest: 8080, host: 8080
  config.vm.network :private_network, ip: "192.168.10.101"

  # Sync this repo into what will be $GOPATH
  config.vm.synced_folder ".", "/opt/go/src/github.com/drone/drone"

  # system-level initial setup
  config.vm.provision "shell", inline: <<-EOF
    set -e

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
    make godep
    export GOPATH=`godep path`:$GOPATH
    export PATH=$PATH:$GOPATH/bin:`godep path`/bin
    make embed
    make build


    # Auto cd to drone install dir
    echo "cd /opt/go/src/github.com/drone/drone" >> /home/vagrant/.bashrc


    # Cleanup
    sudo apt-get autoremove


    echo <<DONE
PROVISIONING COMPLETE:
    vagrant ssh
    make run
    Visit http://localhost:8080/install
DONE
  EOF

  config.vm.provision "shell", inline: <<-EOF
    sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 36A1D7869245C8950F966E92D8576A8BA88D21E9
    sudo sh -c 'echo deb http://get.docker.io/ubuntu docker main' > /etc/apt/sources.list.d/docker.list
    sudo apt-get update
    sudo apt-get install -y linux-image-generic-lts-raring linux-headers-generic-lts-raring
    sudo apt-get install -y lxc-docker-0.9.0
  EOF


end
