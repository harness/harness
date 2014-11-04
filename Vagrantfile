# -*- mode: ruby -*-
# vi: set ft=ruby :

# Vagrantfile API/syntax version. Don't touch unless you know what you're doing!
VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  # Forward keys from SSH agent rather than copypasta
  config.ssh.forward_agent = true

  # FIXME: Maybe this is enough
  config.vm.provider "virtualbox" do |v|
    v.customize ["modifyvm", :id, "--memory", "2048"]
  end

  # Sync this repo into what will be $GOPATH
  config.vm.synced_folder ".", "/opt/go/src/github.com/drone/drone"

  deb_docker = <<-EOF
    sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 36A1D7869245C8950F966E92D8576A8BA88D21E9
    sudo sh -c 'echo deb http://get.docker.io/ubuntu docker main' > /etc/apt/sources.list.d/docker.list
    sudo apt-get update
    sudo apt-get install -y lxc-docker-1.3.0
  EOF

  go = <<-EOF
    set -e
    # Install Go
    GOVERSION="1.3.3"
    GOTARBALL="go${GOVERSION}.linux-amd64.tar.gz"
    export GOROOT=/usr/local/go
    export GOPATH=/opt/go
    export PATH=$PATH:$GOROOT/bin:$GOPATH/bin

    echo "Installing Go $GOVERSION"
    if [ ! $(which go) ]; then
        echo "    Downloading $GOTARBALL"
        wget --quiet --directory-prefix=/tmp http://golang.org/dl/$GOTARBALL

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
    chown -R vagrant:vagrant /opt/go

    # Auto cd to drone install dir
    echo "cd /opt/go/src/github.com/drone/drone" >> /home/vagrant/.bashrc
  EOF

  common = <<-EOF
    npm -g install uglify-js less autoprefixer
    gem install fpm --no-rdoc --no-ri
  EOF

  conf = <<-EOF
    mkdir -p /home/vagrant/.drone/
    cp -R /opt/go/src/github.com/drone/drone/packaging/root/etc/drone/drone.toml /home/vagrant/.drone/config.toml
    sed -i 's;port=":80";port=":8080";g' /home/vagrant/.drone/config.toml
    sed -i 's;/var/lib/drone/drone.sqlite;/opt/go/src/github.com/drone/drone/drone.sqlite;g' /home/vagrant/.drone/config.toml
  EOF

  config.vm.define :ubuntu, primary: true do |ubuntu|
    ubuntu.vm.box = "ubuntu/trusty64"
    ubuntu.vm.network :forwarded_port, guest: 8080, host: 8080
    ubuntu.vm.network :private_network, ip: "192.168.10.101"
    ubuntu.vm.provision "shell", inline: <<-EOF
      echo "Installing Base Packages"
      export DEBIAN_FRONTEND=noninteractive
      sudo add-apt-repository ppa:chris-lea/node.js -y
      sudo apt-get update -qq
      sudo apt-get install -qqy --force-yes build-essential bzr git mercurial vim rpm zip ruby ruby-dev nodejs
    EOF
    ubuntu.vm.provision "shell", inline: deb_docker
    ubuntu.vm.provision "shell", inline: <<-EOF
      sudo apt-get install -y linux-image-generic-lts-trusty linux-headers-generic-lts-trusty
    EOF
    ubuntu.vm.provision "shell", inline: go
    ubuntu.vm.provision "shell", inline: common
    ubuntu.vm.provision "shell", inline: conf

    ubuntu.vm.post_up_message = <<-EOF

      Your machine is up and running

      vagrant reload ubuntu
      vagrant ssh ubuntu
      make run

      http://127.0.0.1:8080
    EOF
  end

  config.vm.define :centos, autostart: false do |centos|
    centos.vm.box = "chef/centos-7.0"
    centos.vm.network :forwarded_port, guest: 8080, host: 8081
    centos.vm.network :private_network, ip: "192.168.10.102"
    centos.vm.provision "shell", inline: <<-EOF
      curl -sL https://rpm.nodesource.com/setup | bash -
      sudo yum install gcc gcc-c++ kernel-devel make bzr git mercurial vim zip ruby ruby-devel nodejs -y
      sudo yum install docker -y
      sudo systemctl enable docker
      sudo systemctl start docker
    EOF
    centos.vm.provision "shell", inline: go
    centos.vm.provision "shell", inline: <<-EOF
      echo "export PATH=$PATH:/usr/local/bin" | sudo tee --append /etc/profile.d/ruby.sh
    EOF
    centos.vm.provision "shell", inline: common
    centos.vm.provision "shell", inline: conf

    centos.vm.post_up_message = <<-EOF

      Your machine is up and running

      vagrant ssh centos
      make run

      http://127.0.0.1:8081
    EOF
  end

  config.vm.define :debian, autostart: false do |debian|
    debian.vm.box = "chef/debian-7.7"
    debian.vm.network :forwarded_port, guest: 8080, host: 8082
    debian.vm.network :private_network, ip: "192.168.10.103"
    debian.vm.provision "shell", inline: <<-EOF
      echo "Installing Base Packages"
      export DEBIAN_FRONTEND=noninteractive
      sudo apt-get update -qq
      sudo apt-get install -qqy --force-yes build-essential bzr git mercurial vim rpm zip ruby ruby-dev curl
      sudo curl -sL https://deb.nodesource.com/setup | bash -
      apt-get install -y nodejs
    EOF
    debian.vm.provision "shell", inline: <<-EOF
      echo "deb http://http.debian.net/debian wheezy-backports main" | sudo tee --append /etc/apt/sources.list > /dev/null
      sudo apt-get update -qq
      sudo apt-get install -yt wheezy-backports linux-image-amd64
    EOF
    debian.vm.provision "shell", inline: deb_docker
    debian.vm.provision "shell", inline: go
    debian.vm.provision "shell", inline: common
    debian.vm.provision "shell", inline: conf

    debian.vm.post_up_message = <<-EOF

      Your machine is up and running

      vagrant ssh debian
      make run

      http://127.0.0.1:8082
    EOF
  end
end