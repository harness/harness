Drone is a Continuous Integration platform built on Docker

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
$ dpkg -i drone.deb
$ sudo start drone
```

Once Drone is running (by default on :80) navigate to **http://localhost:80/install**
and follow the steps in the setup wizard.

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
service:
  - redis
notify:
  email:
    recipients:
      - brad@drone.io
      - burke@drone.io
```

### Environment

Drone clones your repository into a Docker container
at the following location:

```
/var/cache/drone/src/github.com/$owner/$name
```

Please take this into consideration when setting up your build image. For example,
you may need set the $GOAPTH or other environment variables appropriately.

### Databases

Drone can launch database containers for your build: 

```
service:
  - cassandra
  - couchdb
  - elasticsearch
  - neo4j
  - mongodb
  - mysql
  - postgres
  - rabbitmq
  - redis
  - riak
  - zookeeper
```

**NOTE:** database and service containers are exposed over TCP connections and
have their own local IP address. If the **socat** utility is installed inside your
Docker image, Drone will automatically proxy localhost connections to the correct
IP address.

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

```

### Notifications

Drone can trigger email, hipchat and web hook notification at the completion
of your build:

```
notify:
  email:
    recipients:
      - brad@drone.io
      - burke@drone.io

  urls:
    - http://my-deploy-hook.com

  hipchat:
    room: support
	token: 3028700e5466d375
```

### Docs

Coming Soon to [drone.readthedocs.org](http://drone.readthedocs.org/)


