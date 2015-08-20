# Ubuntu

These are the instructions for running Drone on Ubuntu . We recommend using Ubuntu 14.04, the latest stable distribution. We also highly recommend using AUFS as the default file system driver for Docker.

## System Requirements

The default Drone installation uses a SQLite3 database for persistence. Please ensure you have libsqlite3-dev installed:

```
sudo apt-get update
sudo apt-get install libsqlite3-dev
```

The default Drone installation also assumes Docker is installed locally on the host machine. To install Docker on Ubuntu please see the official [installation guide](https://docs.docker.com/installation/ubuntulinux/).

## Installation

Once the environment is prepared you can install Drone from a debian file:

```
wget downloads.drone.io/0.4.0/drone.deb
sudo dpkg --install drone.deb
```

## Settings

Drone uses environment variables for runtime settings and configuration, such as GitHub, GitLab, plugins and more. These settings are loaded from `/etc/drone/dronerc`.

## Starting, Stopping, Logs

Commands to start, stop and restart Drone:

```
sudo start drone
sudo stop drone
sudo restart drone
```

And to view the Drone logs:

```
sudo cat /var/log/upstart/drone.log
```
