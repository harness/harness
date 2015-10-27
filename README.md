[![Build Status](http://beta.drone.io/api/badges/drone/drone/status.svg)](http://beta.drone.io/drone/drone)

Drone is a Continuous Integration platform built on container technology. Every build is executed inside an ephemeral Docker container, giving developers complete control over their build environment with guaranteed isolation.

### Goals

Drone's prime directive is to help teams [ship code like GitHub](https://github.com/blog/1241-deploying-at-github#always-be-shipping). Drone is easy to install, setup and maintain and offers a powerful container-based plugin system. Drone aspires to be an industry-wide replacement for Jenkins.

### Documentation

Drone documentation is organized into several categories:

* [Setup Guide](http://readme.drone.io/setup/)
* [Build Guide](http://readme.drone.io/build/)
* [Plugin Guide](http://readme.drone.io/plugin/)
* [CLI Reference](http://readme.drone.io/cli/)
* [API Reference](http://readme.drone.io/api/)

### Community, Help

Contributions, questions, and comments are welcomed and encouraged. Drone developers hang out in the [drone/drone](https://gitter.im/drone/drone) room on gitter. We ask that you please post your questions to [gitter](https://gitter.im/drone/drone) before creating an issue.

### Cloning, Building, Running

If you are new to Go, make sure you [install](http://golang.org/doc/install) Go 1.5+ and [setup](http://golang.org/doc/code.html) your workspace (ie `$GOPATH`). Go programs use directory structure for package imports, therefore, it is very important you clone this project to the specified directory in your Go path:

```
git clone git://github.com/drone/drone.git $GOPATH/src/github.com/drone/drone
cd $GOPATH/src/github.com/drone/drone
```

Please ensure your local environment has the following dependencies installed. We provide scripts in the `./contrib` folder as a convenience that can be used to install:

* libsqlite3
* sassc

Commands to build from source:

```sh
export GO15VENDOREXPERIMENT=1

make deps    # Download required dependencies
make gen     # Generate code
make build   # Build the binary
```

Commands for development:

```sh
make gen_static     # Generate static content
make gen_template   # Generate templates from amber files
make gen_migrations # Generate embedded database migrations
make vet            # Execute go vet command
make fmt            # Execute go fmt command
```

Commands to start drone:

```sh
drone
drone --debug # Debug mode enables more verbose logging
```

If you are seeing slow compile times please install the following:

```sh
go install github.com/mattn/go-sqlite3
```
