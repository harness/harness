These are the instructions for running Drone on Ubuntu . We recommend using Ubuntu 14.04, the latest stable distribution. We also highly recommend using AUFS as the default file system driver for Docker.

### System Requirements

The default Drone installation uses a SQLite3 database for persistence. Please ensure you have `libsqlite3-dev` installed:

```bash
sudo apt-get update
sudo apt-get install libsqlite3-dev
```

The default Drone installation also assumes Docker is installed locally on the host machine. To install Docker on Ubuntu:

```bash
wget -qO- https://get.docker.com/ | sh
```

### Installation

Once the environment is prepared you can install Drone from a debian file. Drone will automatically start on port 80. Edit /etc/drone/drone.toml to modify the port.

```bash
wget downloads.drone.io/master/drone.deb
sudo dpkg --install drone.deb
```

### Start & Stop

The Drone service is managed by upstart and can be started, stopped or restarted using the following commands:

```bash
sudo start drone
sudo stop drone
sudo restart drone
```

### Logging

The Drone service logs are written to `/var/log/upstart/drone.log`.