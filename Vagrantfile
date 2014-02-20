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
      v.customize ["modifyvm", :id, "--memory", "1024"]
  end

  # Drone by default runs on port 80. Forward from host to guest
  config.vm.network :forwarded_port, guest: 80, host: 8080
  config.vm.network :private_network, ip: "192.168.56.101"

  # system-level initial setup
  config.vm.provision "shell" do |s|
    s.path = "scripts/provision.sh"
  end
end
