[![Build Status](http://test.drone.io/api/badge/github.com/drone/drone/status.svg?style=flat)](http://test.drone.io/github.com/drone/drone)
[![GoDoc](https://godoc.org/github.com/drone/drone?status.svg)](https://godoc.org/github.com/drone/drone)
[![Gitter](https://badges.gitter.im/Join Chat.svg)](https://gitter.im/drone/drone?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

## Documentation

* [Installation](http://readme.drone.io/setup/install/ubuntu/)
* [User Guide](http://readme.drone.io/usage/overview/)
* [API Reference](http://readme.drone.io/api/overview/)

## System Requirements

* Docker
* AUFS


We highly recommend running Docker with the AUFS storage driver. You can verify Docker is using
the AUFS storage driver with the following command `sudo docker info | grep Driver:`

## Upgrading

If you are upgrading from `0.2` I would recommend waiting a few weeks for the master
branch to stabilize. There was a huge amount of refactoring that destabilized the codebase
and I'd hate for that to impact any real world installations.

If you still want to upgrade to `0.2` please know that the databases are not compatible and
there is no automated migration due to some fundamental structural changes. You will need
to start with a fresh instance.

## Installation

**This is project is alpha stage. Consider yourself warned**

We have optimized the installation process for Ubuntu since that is what we test with internally.
You can run the following commands to quickly download an install Drone on an Ubuntu machine.

```sh
# Ubuntu, Debian
wget downloads.drone.io/master/drone.deb
sudo dpkg -i drone.deb

# CentOS, RedHat
wget downloads.drone.io/master/drone.rpm
sudo yum localinstall drone.rpm
```

## Database

By default, Drone will create a SQLite database. Drone also supports Postgres and MySQL
databases. You can customize the database settings using the configuration options
described in the **Setup** section.

Below are some example configurations that you can use as reference:

```toml
# to use postgres
[database]
driver="postgres"
datasource="host=127.0.0.1 user=postgres dbname=drone sslmode=disable"

# to use mysql
[database]
driver="mysql"
datasource="root@tcp(127.0.0.1:3306)/drone"
```

## Setup

We are in the process of moving configuration out of the UI and into configuration
files and/or environment variables (your choice which). If you prefer configuration files
you can provide Drone with the path to your configuration file:

```sh
droned --config=/path/to/drone.toml
```

The configuration file is in TOML format. If installed using the `drone.deb` file
will be located in `/etc/drone/drone.toml`.

You can find the current config of the master branch [here](https://github.com/drone/drone/blob/master/packaging/root/etc/drone/drone.toml).

```toml

[server]
port=""

[server.ssl]
key=""
cert=""

[session]
secret=""
expires=""

[database]
driver=""
datasource=""

[github]
client=""
secret=""
orgs=[]
open=false

[github_enterprise]
client=""
secret=""
api=""
url=""
orgs=[]
private_mode=false
open=false

[bitbucket]
client=""
secret=""
open=false

[gitlab]
url=""
client=""
secret=""
skip_verify=false
open=false

[gogs]
url=""
secret=""
open=false

[smtp]
host=""
port=""
from=""
user=""
pass=""

[docker]
cert=""
key=""

[worker]
nodes=[
"unix:///var/run/docker.sock",
"unix:///var/run/docker.sock"
]

```

Or you can use environment variables

```sh

# custom http server settings
export DRONE_SERVER_PORT=""
export DRONE_SERVER_SSL_KEY=""
export DRONE_SERVER_SSL_CERT=""

# session settings
export DRONE_SESSION_SECRET=""
export DRONE_SESSION_EXPIRES=""

# custom database settings
export DRONE_DATABASE_DRIVER=""
export DRONE_DATABASE_DATASOURCE=""

# github configuration
export DRONE_GITHUB_CLIENT=""
export DRONE_GITHUB_SECRET=""
export DRONE_GITHUB_OPEN=false

# github enterprise configuration
export DRONE_GITHUB_ENTERPRISE_CLIENT=""
export DRONE_GITHUB_ENTERPRISE_SECRET=""
export DRONE_GITHUB_ENTERPRISE_API=""
export DRONE_GITHUB_ENTERPRISE_URL=""
export DRONE_GITHUB_ENTERPRISE_PRIVATE_MODE=false
export DRONE_GITHUB_ENTERPRISE_OPEN=false

# bitbucket configuration
export DRONE_BITBUCKET_CLIENT=""
export DRONE_BITBUCKET_SECRET=""
export DRONE_BITBUCKET_OPEN=false


# gitlab configuration
export DRONE_GITLAB_URL=""
export DRONE_GITLAB_CLIENT=""
export DRONE_GITLAB_SECRET=""
export DRONE_GITLAB_SKIP_VERIFY=false
export DRONE_GITLAB_OPEN=false

# email configuration
export DRONE_SMTP_HOST=""
export DRONE_SMTP_PORT=""
export DRONE_SMTP_FROM=""
export DRONE_SMTP_USER=""
export DRONE_SMTP_PASS=""

# worker nodes
# these are optional. If not specified Drone will add
# two worker nodes that connect to $DOCKER_HOST
export DRONE_WORKER_NODES="tcp://0.0.0.0:2375,tcp://0.0.0.0:2375"
```

Or a combination of the two:

```sh
DRONE_GITLAB_URL="https://gitlab.com" droned --config=/path/to/drone.conf
```

## GitHub

In order to setup with GitHub you'll need to register your local Drone installation
with GitHub (or GitHub Enterprise). You can read more about registering an application here:
https://github.com/settings/applications/new

Below are example values when running Drone locally. If you are running Drone on a server
you should replace `localhost` with your server hostname or address.

Homepage URL:

```
http://localhost:8000/
```

Authorization callback URL:

```
http://localhost:8000/api/auth/github.com
```

## Build Configuration

You will need to include a `.drone.yml` file in the root of your repository in order to
configure a build. I'm still working on updated documentation, so in the meantime please refer
to the `0.2` README to learn more about the `.drone.yml` format:

https://github.com/drone/drone/blob/v0.2.1/README.md#builds
