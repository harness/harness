[![Build Status](http://test.drone.io/v1/badge/github.com/drone/drone/status.svg?branch=exp)](http://test.drone.io/github.com/drone/drone)
[![GoDoc](https://godoc.org/github.com/drone/drone?status.png)](https://godoc.org/github.com/drone/drone)


## System Requirements

* Docker
* AUFS

We highly recommend running Docker with the AUFS storage driver. You can verify Docker is using
the AUFS storage driver with the following command `sudo docker info | grep Driver:`

## Installation

We have optimized the installation process for Ubuntu since that is what we test with internally. You can run the following commands to quickly download an install Drone on an Ubuntu machine.

```sh
wget downloads.drone.io/exp/drone.deb
sudo dpkg -i drone.deb
```

## Setup

We are in the process of moving configuration out of the UI and into configuration
files and/or environment variables (your choice which). If you prefer configuration files
you can provide Drone with the path to your configuration file:

```sh
./drone --config=/path/to/drone.conf
```

The configuration file is in TOML format:

```toml

[registration]
open=true

[github]
client=""
secret=""

[github_enterprise]
client=""
secret=""
api=""
url=""
private_mode=false

[bitbucket]
client=""
secret=""

[gitlab]
url=""

[smtp]
host=""
port=""
from=""
user=""
pass=""

[worker]
nodes=[
"unix:///var/run/docker.sock",
"unix:///var/run/docker.sock"
]

```

Or you can use environment variables

```sh

# enable users to self-register
export DRONE_REGISTRATION_OPEN=false

# github configuration
export DRONE_GITHUB_CLIENT=""
export DRONE_GITHUB_SECRET=""

# github enterprise configuration
export DRONE_GITHUB_ENTERPRISE_CLIENT=""
export DRONE_GITHUB_ENTERPRISE_SECRET=""
export DRONE_GITHUB_ENTERPRISE_API=""
export DRONE_GITHUB_ENTERPRISE_URL=""
export DRONE_GITHUB_ENTERPRISE_PRIVATE_MODE=false

# bitbucket configuration
export DRONE_BITBUCKET_CLIENT=""
export DRONE_BITBUCKET_SECRET=""

# gitlab configuration
export DRONE_GITLAB_URL=""

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
DRONE_GITLAB_URL="https://gitlab.com" ./drone --config=/path/to/drone.conf
```

### SQLite

```toml
[database]
driver="sqlite3"
datasurce="/var/lib/drone/drone.sqlite"
```

Or you can use environment variables

```sh
DRONE_DATABASE_DRIVER="sqlite3"
DRONE_DATABASE_DATASOURCE="/var/lib/drone/drone.sqlite"
```

### PostgreSQL
More information about data source you can read [here](http://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters)

**TCP**

```toml
[database]
driver="postgres"
datasurce="user=postgres password=drone host=127.0.0.1 port=5432 dbname=drone sslmode=disable"
```

Or you can use environment variables

```sh
DRONE_DATABASE_DRIVER="postgres"
DRONE_DATABASE_DATASOURCE="user=postgres password=drone host=127.0.0.1 port=5432 dbname=drone sslmode=disable"
```

**UNIX**

```toml
[database]
driver="postgres"
datasurce="user=postgres password=drone host=/var/run/postgresql/.s.PGSQL.5432 dbname=drone sslmode=disable"
```

Or you can use environment variables

```sh
DRONE_DATABASE_DRIVER="postgres"
DRONE_DATABASE_DATASOURCE="user=postgres password=drone host=/var/run/postgresql/.s.PGSQL.5432 dbname=drone sslmode=disable"
```

### MySQL
More information about data source you can read [here](https://github.com/go-sql-driver/mysql#dsn-data-source-name)

**TCP**

```toml
[database]
driver="mysql"
datasurce="drone:drone@tcp(127.0.0.1:3306)/drone"
```

Or you can use environment variables

```sh
DRONE_DATABASE_DRIVER="mysql"
DRONE_DATABASE_DATASOURCE="drone:drone@tcp(127.0.0.1:3306)/drone"
```

**UNIX**

```toml
[database]
driver="mysql"
datasurce="drone:drone@unix(/tmp/mysql.sock)/drone"
```

Or you can use environment variables

```sh
DRONE_DATABASE_DRIVER="mysql"
DRONE_DATABASE_DATASOURCE="drone:drone@unix(/tmp/mysql.sock)/drone"
```

## Compatibility Issues

**WARNING**

There were some fundamental changes to the application and we decided to introduce breaking changes to the dataabase. Migration would have been difficult and time consuming. Drone is an alpha product and therefore backward compatibility is not a primary goal until we hit a stable release. Apologizes for any inconvenience.
