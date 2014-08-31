[![Build Status](http://beta.drone.io/github.com/drone/drone/status.svg?branch=exp)](http://beta.drone.io/github.com/drone/drone?branch=exp)
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
https://github.com/drone/drone/milestones/v0.3

## What Changed

This branch introduces major changes, including:

1. modification to project structure
2. api-driven design
3. interface to abstract github, bitbucket, gitlab code (see /shared/remote)
4. handlers are structures with service providers "injected"
5. github, bitbucket, etc native permissions are used. No more teams or permissions in Drone
6. github, bitbucket, etc authentication is used. No more drone password
7. github, bitbucket, etc repository data is cached upon login (and subsequent logins)
8. angularjs user interface with modified responsive design

## Database 

For Debian, database configs stores in /etc/default/drone

### SQLite

```sh
DRONE_DRIVER="sqlite3"
DRONE_DATASOURCE="/var/lib/drone/drone.sqlite"
```

### PostgreSQL
More information about data source you can read [here](http://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters)

**TCP**

```sh
DRONE_DRIVER="postgres"
DRONE_DATASOURCE="user=postgres password=drone host=127.0.0.1 port=5432 dbname=drone sslmode=disable"
```

**UNIX**

```sh
DRONE_DRIVER="postgres"
DRONE_DATASOURCE="user=postgres password=drone host=/var/run/postgresql/.s.PGSQL.5432 dbname=drone sslmode=disable"
```

### MySQL
More information about data source you can read [here](https://github.com/go-sql-driver/mysql#dsn-data-source-name)

**TCP**

```sh
DRONE_DRIVER="mysql"
DRONE_DATASOURCE="drone:drone@tcp(127.0.0.1:3306)/drone"
```

**UNIX**

```sh
DRONE_DRIVER="mysql"
DRONE_DATASOURCE="drone:drone@unix(/tmp/mysql.sock)/drone"
```

### Compatibility Issues

**WARNING**

There were some fundamental changes to the application and we decided to introduce breaking changes to the dataabase. Migration would have been difficult and time consuming. Drone is an alpha product and therefore backward compatibility is not a primary goal until we hit a stable release. Apologizes for any inconvenience.

## Filing Bugs

This is an experimental branch with known bugs and issues, namely:

* notifications
* github status api updates
* gitlab integration
* smtp settings screen
* github / bitbucket / gitlab settings screen
* mysql support

Please do not log issues for the above items. We are aware and will fix as soon as possible, as they are holding up the 0.3 release. Pull requests, however, are very welcome :)

You can track progress of this release here:

https://github.com/drone/drone/milestones/v0.3
