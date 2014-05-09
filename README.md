Drone is a [Continuous Integration](http://en.wikipedia.org/wiki/Continuous_integration) platform built on [Docker](https://www.docker.io/)

[![Build Status](http://beta.drone.io/github.com/drone/drone/status.svg?branch=master)](http://beta.drone.io/github.com/drone/drone)
[![GoDoc](https://godoc.org/github.com/drone/drone?status.png)](https://godoc.org/github.com/drone/drone)

### Getting Started

* [Installation](http://drone.readthedocs.org/en/latest/install.html)
* [Configuration](http://drone.readthedocs.org/en/latest/setup.html)
* [Getting Help](http://drone.readthedocs.org/en/latest/index.html#need-help)

### Contributing

Interested in contributing? Great! Please read our [contributor guidelines](http://drone.readthedocs.org/en/latest/contribute.html#pull-requests).

---

* [System Requirements](#system)
* [Installation](#setup)
* [Builds](#builds)
* [Images](#images)
* [Application Environment](#environment)
* [Git Command Options](#git-command-options)
* [Deployments](#deployments)
* [Notifications](#notifications)
* [Database Services](#databases)
* [Caching](#caching)
* [Params Injection](#params-injection)
* [Wall display](#wall-display)
* [Local development](#local-development)
* [Documentation and References](#docs)

### System

Drone is tested on the following versions of Ubuntu:

* Ubuntu Precise 12.04 (LTS) (64-bit)
* Ubuntu Raring 13.04 (64 bit)

Drone's only external dependency is the latest version of Docker (0.8)

### Setup

Drone is packaged and distributed as a debian file. You can download an install
using the following commands:

```sh
$ wget http://downloads.drone.io/latest/drone.deb
$ sudo dpkg -i drone.deb
$ sudo start drone
```

Once Drone is running (by default on :80) navigate to **http://localhost:80/install**
and follow the steps in the setup wizard.

**IMPORTANT** You will also need a GitHub Client ID and Secret:

* Register a new application https://github.com/settings/applications
* Set the homepage URL to http://$YOUR_IP_ADDRESS/
* Set the callback URL to http://$YOUR_IP_ADDRESS/auth/login/github
* Copy the Client ID and Secret into the Drone admin console http://localhost:80/account/admin/settings

I'm working on a getting started video. Having issues with volume, but hopefully
you can still get a feel for the steps:

https://docs.google.com/file/d/0By8deR1ROz8memUxV0lTSGZPQUk

**Using MySQL**

By default, Drone use sqlite as its database storage. To use MySQL/MariaDB instead, use `-driver` flag
and set it to `mysql`. You will need to set your DSN (`-datasource`) in this form: 
```
    user:password@tcp(hostname:port)/dbname?parseTime=true
```
Change it according to your database settings. The parseTime above is required since drone using
`time.Time` to represents `TIMESTAMP` data. Please refer to [1] for more options on mysql driver.

You may also need to tweak some innodb options, especially if you're using `utf8mb4` collation type.
```
    innodb_file_format = Barracuda
    innodb_file_per_table = On
    innodb_large_prefix = On
```
Please consult to the MySQL/MariaDB documentation for further information
regarding large prefix for index column and dynamic row format (which is used in Drone).

[1] https://github.com/go-sql-driver/mysql

### Builds

Drone use a **.drone.yml** configuration file in the root of your
repository to run your build:

```
image: mischief/docker-golang
env:
  - GOPATH=/var/cache/drone
script:
  - go build
  - go test -v
services:
  - redis
notify:
  email:
    recipients:
      - brad@drone.io
      - burke@drone.io
```

### Images

In the above example we used a custom Docker image from index.docker.io **mischief/docker-golang**

Drone also provides official build images. These images are configured specifically for CI and
have many common software packages pre-installed (git, xvfb, firefox, libsqlite, etc).

Official Drone images are referenced in the **.drone.yml** by an alias:

```sh
image: go1.2   # same as bradrydzewski/go:1.2
```

Here is a list of our official images:

```sh
# these two are base images. all Drone images are built on top of these
# these are BIG (~3GB) so make sure you have a FAST internet connection
docker pull bradrydzewski/ubuntu
docker pull bradrydzewski/base

# clojure images
docker pull bradrydzewski/lein             # image: lein

# dart images
docker pull bradrydzewski/dart:stable      # image: dart

# erlang images
docker pull bradrydzewski/erlang:R16B      # image: erlangR16B
docker pull bradrydzewski/erlang:R16B02    # image: erlangR16B02
docker pull bradrydzewski/erlang:R16B01    # image: erlangR16B01

# gcc images (c/c++)
docker pull bradrydzewski/gcc:4.6          # image: gcc4.6
docker pull bradrydzewski/gcc:4.8          # image: gcc4.8

# go images
docker pull bradrydzewski/go:1.0           # image: go1
docker pull bradrydzewski/go:1.1           # image: go1.1
docker pull bradrydzewski/go:1.2           # image: go1.2

# haskell images
docker pull bradrydzewski/haskell:7.4      # image: haskell

# java and jdk images
docker pull bradrydzewski/java:openjdk6    # image: openjdk6
docker pull bradrydzewski/java:openjdk7    # image: openjdk7
docker pull bradrydzewski/java:oraclejdk7  # image: oraclejdk7
docker pull bradrydzewski/java:oraclejdk8  # image: oraclejdk8

# node images
docker pull bradrydzewski/node:0.10        # image node0.10
docker pull bradrydzewski/node:0.8         # image node0.8

# php images
docker pull bradrydzewski/php:5.5          # image: php5.5
docker pull bradrydzewski/php:5.4          # image: php5.4

# python images
docker pull bradrydzewski/python:2.7       # image: python2.7
docker pull bradrydzewski/python:3.2       # image: python3.2
docker pull bradrydzewski/python:3.3       # image: python3.3
docker pull bradrydzewski/python:pypy      # image: pypy

# ruby images
docker pull bradrydzewski/ruby:2.0.0       # image: ruby2.0.0
docker pull bradrydzewski/ruby:1.9.3       # image: ruby1.9.3

# scala images
docker pull bradrydzewski/scala:2.10.3     # image: scala2.10.3
docker pull bradrydzewski/scala:2.9.3      # image: scala2.9.3

```

### Environment

Drone clones your repository into a Docker container
at the following location:

```
/var/cache/drone/src/github.com/$owner/$name
```

Please take this into consideration when setting up your build commands, or
if you are using a custom Docker image.

### Git Command Options

You can specify the `--depth` option of the `git clone` command (default value is `50`):

```
git:
  depth: 1
```

### Deployments

Drone can trigger a deployment at the successful completion of your build:

```
deploy:
  heroku:
    app: safe-island-6261

publish:
  s3:
    acl: public-read
    region: us-east-1
    bucket: downloads.drone.io
    access_key: C24526974F365C3B
    secret_key: 2263c9751ed084a68df28fd2f658b127
    source: /tmp/drone.deb
    target: latest/

  swift:
    username: someuser
    password: 030e39a1278a18828389b194b93211aa
    auth_url: https://identity.api.rackspacecloud.com/v2.0
    region: DFW
    container: drone
    source: /tmp/drone.deb
    target: latest/drone.deb
    branch: master

```

Drone currently has these `deploy` and `publish` plugins implemented (more to come!):

**deploy**
- [heroku](#docs)
- [git](#docs)
- [modulus](#docs)
- [nodejitsu](#docs)
- [ssh](#docs)
- [tsuru](#docs)
- [bash](#docs)

**publish**
- [Amazon s3](#docs)
- [OpenStack Swift](#docs)
- [PyPI](#docs)

Publish plugins can be limited to a specific branch using the `branch` configuration
as seen above in the `swift` example. If you do not specify a `branch` all branches
will be published, with the exception of Pull Requests.

### Notifications

Drone can trigger email, hipchat and web hook notification at the beginning and
completion of your build:

```
notify:
  email:
    recipients:
      - brad@drone.io
      - burke@drone.io

  webhook:
    on_success: true
    on_failure: true
    urls:
      - http://my-deploy-hook.com

  hipchat:
    room: support
    token: 3028700e5466d375
    on_started: true
    on_success: true
    on_failure: true
```

### Databases

Drone can launch database containers for your build:

```
services:
  - cassandra
  - couchdb
  - couchdb:1.0
  - couchdb:1.4
  - couchdb:1.5
  - elasticsearch
  - elasticsearch:0.20
  - elasticsearch:0.90
  - neo4j
  - neo4j:1.9
  - mongodb
  - mongodb:2.2
  - mongodb:2.4
  - mysql
  - mysql:5.5
  - postgres
  - postgres:9.1
  - rabbitmq
  - rabbitmq:3.2
  - redis
  - riak
  - zookeeper
```

If you omit the version, Drone will launch the latest version of the database. (For example, if you set `mongodb`, Drone will launch MongoDB 2.4.)

You can also launch custom Docker containers using standard docker notation:

```sh
services:
  - dockerfile/rethinkdb # same as dockerfile/rethinkdb:latest
  - barnybug/elasticsearch:1.0.1
```

**NOTE 1:** database and service containers are exposed over TCP connections and
have their own local IP address. If the **socat** utility is installed inside your
Docker image, Drone will automatically proxy localhost connections to the correct
IP address.

**NOTE 2:** avoid running services that use the same ports. For example, don't specify
multiple versions of Elastic Search since the port will already be in use.

### Caching

Drone can persist directories between builds. This should be used for caching dependencies to
decrease overall build time. Examples include your `.npm`, `.m2`, `bundler`, etc.

```
cache:
  - /usr/local/bin/go/pkg
```

This will cache the directory relative to the root directory of your repository:

```
cache:
  - .npm
```

**NOTE:** this is an alpha quality feature and still has some quirks. See https://github.com/drone/drone/issues/147

### Params Injection

You can inject params into .drone.yml.

```
notify:
  hipchat:
    room: {{hipchatRoom}}
    token: {{hipchatToken}}
    on_started: true
    on_success: true
    on_failure: true
```

![params-injection](https://f.cloud.github.com/assets/1583973/2161187/2905077e-94c3-11e3-8499-a3844682c8af.png)

### Wall display

A wall display for Drone is available as a separate service. See [Drone Wall](https://github.com/drone/drone-wall)
for details.

### Local development

Local Drone setup for development is pretty straightforward.

You will need to clone the repo, install Vagrant and run `vagrant up`.
This command will download base Ubuntu image, setup the virtual machine and build Drone.

Afterwards, you may `vagrant ssh` into the vagrant instance, where docker is already installed and ready to go.

Once in the vagrant instance, run `make run`, the visit http://localhost:8080/install in your browser.

The Makefile has other targets so check that out for more build, test, run configurations.

### Docs

* [drone.readthedocs.org](http://drone.readthedocs.org/) (Coming Soon)
* [GoDoc](http://godoc.org/github.com/drone/drone)

